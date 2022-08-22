package shortener_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/usa4ev/urlshortner/internal/storage"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/shortener"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ctXML  string = "application/xml"
	ctJSON string = "application/json"
	ctText string = "text/plain"
)

type (
	test struct {
		url  string
		want string
	}
	tests []test
)

func newTestClient(ts *httptest.Server) *http.Client {
	cl := ts.Client()

	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	return cl
}

func newTestSrv(srvAddr string) *httptest.Server {
	c := testConfig()
	s := shortener.NewShortener(c, storage.New(c))
	r := router.NewRouter(s)

	l, err := net.Listen("tcp", srvAddr)
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen on %v: %v", "localhost:8080", err))
	}

	ts := httptest.NewUnstartedServer(r)
	ts.Listener = l
	ts.Start()

	return ts
}

func getTests(baseURL string) tests {
	urls := []string{
		"http://ya.ru/test",
		"http://vk.com/test/test2",
		"https://xn--80aesfpebagmfblc0a.xn--p1ai/test/test2/",
	}
	res := tests{}

	for _, v := range urls {
		res = append(res, test{v, baseURL + "/" + base64.RawURLEncoding.EncodeToString([]byte(v))})
	}

	return res
}

func Test_MakeShort(t *testing.T) {
	config := testConfig()
	cases := getTests(config.BaseURL())
	ts := newTestSrv(config.SrvAddr())
	cl := newTestClient(ts)
	resetStorage(config.StoragePath(), config.DBDSN())

	defer ts.Close()

	for _, tt := range cases {
		t.Run("POST no-JSON", func(t *testing.T) {
			res, err := cl.Post(ts.URL, ctText, bytes.NewBuffer([]byte(tt.url)))
			require.NoError(t, err, "url: %v", tt.url)

			if res.StatusCode != http.StatusCreated {
				t.Errorf("failed when filling test data\nurl: %v\nstatus:%v", tt.url, res.StatusCode)

				return
			}

			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close())
			assert.Equal(t, tt.want, string(body))
		})
	}
}

func Test_MakeShortJSON(t *testing.T) {
	config := testConfig()
	cases := getTests(config.BaseURL())
	ts := newTestSrv(config.SrvAddr())
	cl := newTestClient(ts)

	defer ts.Close()
	resetStorage(config.StoragePath(), config.DBDSN())

	for _, tt := range cases {
		t.Run("POST JSON", func(t *testing.T) {
			require.NoError(t, resetStorage(config.StoragePath(), config.DBDSN()), "failed to reset storage")

			req := struct {
				URL string `json:"url"`
			}{tt.url}
			w := bytes.NewBuffer(nil)
			enc := json.NewEncoder(w)
			require.NoError(t, enc.Encode(req), "url: %v", tt.url)

			res, err := cl.Post(ts.URL+"/api/shorten", ctJSON, w)
			require.NoError(t, err, "url: %v", tt.url)

			if res.StatusCode != http.StatusCreated {
				t.Errorf("failed when filling test data\nurl: %v\nstatus:%v", tt.url, res.StatusCode)

				return
			}

			message := struct {
				Result string `json:"result"`
			}{}
			dec := json.NewDecoder(res.Body)
			err = dec.Decode(&message)
			if err != nil {
				message, _ := io.ReadAll(res.Body)
				require.NoError(t, err, "failed to parse message:%v", string(message))
			}

			require.NoError(t, res.Body.Close())
			assert.Equal(t, tt.want, message.Result, "result mismatch\nurl: %v\nstatus:%v", tt.url, res.StatusCode)
		})
	}

	t.Run("Wrong content type header", func(t *testing.T) {
		resetStorage(config.StoragePath(), config.DBDSN())
		tt := cases[0]
		req := struct {
			URL string `json:"url"`
		}{tt.url}
		w := bytes.NewBuffer(nil)
		enc := json.NewEncoder(w)
		err := enc.Encode(req)
		require.NoError(t, err, "url: %v", tt.url)

		res, err := cl.Post(ts.URL+"/api/shorten", ctXML, w)
		require.NoError(t, err, "url: %v", tt.url)
		require.NoError(t, res.Body.Close())
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "got wrong status code")
	})

	t.Run("Wrong message format", func(t *testing.T) {
		tt := cases[0]

		res, err := cl.Post(ts.URL+"/api/shorten", ctJSON, bytes.NewBuffer([]byte(tt.url)))
		require.NoError(t, err, "url: %v", tt.url)
		require.NoError(t, res.Body.Close())
		assert.Equal(t, http.StatusBadRequest, res.StatusCode, "got wrong status code")
	})
}

func Test_MakeLong_EmptyStorage(t *testing.T) {
	config := testConfig()
	resetStorage(config.StoragePath(), config.DBDSN())
	cases := getTests(config.BaseURL())

	ts := newTestSrv(config.SrvAddr())
	defer ts.Close()

	t.Run("Empty storage", func(t *testing.T) {
		cl := newTestClient(ts)
		tt := cases[0]
		res, err := cl.Get(tt.want)
		require.NoError(t, err, "url: %v", tt.url)
		require.NoError(t, res.Body.Close())
		require.Equal(t, http.StatusNotFound, res.StatusCode, "got wrong status code\nurl:%v", tt.url)
	})
}

func Test_GetURLsByUser(t *testing.T) {
	config := testConfig()
	cases := getTests(config.BaseURL())
	ts := newTestSrv(config.SrvAddr())
	defer ts.Close()
	cl := newTestClient(ts)
	defer resetStorage(config.StoragePath(), config.DBDSN())

	t.Run("Get URL's by user", func(t *testing.T) {
		var userID string
		for _, tt := range cases {
			req, err := http.NewRequest("POST", ts.URL, bytes.NewBuffer([]byte(tt.url)))
			req.Header = map[string][]string{
				"Accept-Encoding": {"gzip, deflate"},
				"Content-type":    {ctText},
			}
			req.AddCookie(&http.Cookie{Name: "userID", Value: userID})

			res, err := cl.Do(req)
			if userID == "" {
				userID = getUserID(res.Cookies())
			}
			require.NoError(t, err, "url: %v", tt.url)
			require.NoError(t, res.Body.Close(), "url: %v", tt.url)
			require.Equal(t, http.StatusCreated, res.StatusCode)
		}

		req, err := http.NewRequest("GET", ts.URL+"/api/user/urls", nil)
		req.Header = map[string][]string{
			"Accept-Encoding": {"gzip, deflate"},
		}
		req.AddCookie(&http.Cookie{Name: "userID", Value: userID})

		res, err := cl.Do(req)

		if err != nil {
			v, err := io.ReadAll(res.Body)
			require.NoError(t, err, "failed to read error response")
			require.NoError(t, fmt.Errorf(string(v)))
		}

		var message []storage.Pair

		dec := json.NewDecoder(res.Body)
		err = dec.Decode(&message)
		require.NoError(t, res.Body.Close())
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode, "got wrong status code")
		assert.Equal(t, len(cases), len(message), " got wrong number of urls")
	})
}

func Test_MakeLong(t *testing.T) {
	config := testConfig()
	cases := getTests(config.BaseURL())
	ts := newTestSrv(config.SrvAddr())
	defer ts.Close()
	cl := newTestClient(ts)

	t.Run("Get URL", func(t *testing.T) {
		resetStorage(config.StoragePath(), config.DBDSN())
		for _, tt := range cases {
			res, err := cl.Post(ts.URL, ctText, bytes.NewBuffer([]byte(tt.url)))
			require.NoError(t, err, "url: %v", tt.url)
			require.NoError(t, res.Body.Close(), "url: %v", tt.url)
			require.Equal(t, http.StatusCreated, res.StatusCode)
		}

		for _, tt := range cases {
			res, err := cl.Get(tt.want)
			require.NoError(t, err, "url: %v", tt.url)
			response, err := io.ReadAll(res.Body)
			require.NoError(t, res.Body.Close(), "url: %v", tt.url)
			require.NoError(t, err)
			assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode, "got wrong status code\nurl:%v\nresponse:%v", tt.url, string(response))
			assert.Equal(t, tt.url, res.Header.Get("Location"), " got wrong location")
		}
	})

	t.Run("Get nonexistent url", func(t *testing.T) {
		if err := resetStorage(config.StoragePath(), config.DBDSN()); err != nil {
			require.NoError(t, err, "failed to reset storage")
		}
		resetStorage(config.StoragePath(), config.DBDSN())
		tt := cases[0]
		res, err := cl.Get(tt.want + "i")
		require.NoError(t, err, "url: %v", tt.url)
		require.NoError(t, res.Body.Close())
		assert.Equal(t, http.StatusNotFound, res.StatusCode, "got wrong status code\nurl%v:", tt.url)
	})

	tt := cases[0]

	t.Run("Get gzipMW", func(t *testing.T) {
		resetStorage(config.StoragePath(), config.DBDSN())
		res, err := cl.Post(ts.URL, ctText, bytes.NewBuffer([]byte(tt.url)))
		require.NoError(t, err, "url: %v", tt.url)
		require.Equal(t, http.StatusCreated, res.StatusCode)
		require.NoError(t, res.Body.Close(), "url: %v", tt.url)

		res, err = cl.Get(tt.want)
		require.NoError(t, res.Body.Close(), "url: %v", tt.url)
		require.NoError(t, err, "url: %v", tt.url)
		assert.Equal(t, http.StatusTemporaryRedirect, res.StatusCode, "got wrong status code\nurl%v:", tt.url)
	})
}

func getUserID(cc []*http.Cookie) string {
	for _, cookie := range cc {
		if cookie.Name == "userID" {
			return cookie.Value
		}
	}
	return ""
}

func resetStorage(path, dsn string) error {
	// path is not set, quit wo error
	if path == "" {
		return nil
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	fmt.Println("storage reset successful")

	return nil
}

func testConfig() *config.Config {
	return config.New(config.WithEnvVars(map[string]string{
		"BASE_URL":          "http://localhost:8080",
		"SERVER_ADDRESS":    "localhost:8080",
		"FILE_STORAGE_PATH": os.Getenv("HOME") + "/storage.csv",
	}),
		config.IgnoreOsArgs())
}
