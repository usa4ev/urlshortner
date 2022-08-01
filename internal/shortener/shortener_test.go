package shortener

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/usa4ev/urlshortner/internal/configrw"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	host   string = "localhost:8080"
	ctXML  string = "application/xml"
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

func newTestSrv() *httptest.Server {
	r := NewRouter()
	r.Route("/", DefaultRoute())

	l, err := net.Listen("tcp", host)
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen on %v: %v", "localhost:8080", err))
	}

	ts := httptest.NewUnstartedServer(r)
	ts.Listener = l
	ts.Start()

	return ts
}

func getTests() tests {
	baseURL := configrw.ReadBaseURL()
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
	cases := getTests()
	ts := newTestSrv()
	cl := newTestClient(ts)
	defer resetStorage()

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

		/*		t.Run("POST no-JSON gzip", func(t *testing.T) {
				buf := bytes.NewBuffer(nil)
				writer, err := gzip.NewWriterLevel(buf, gzip.BestSpeed)
				require.NoError(t, err, "url: %v", tt.url)
				_, err = writer.Write([]byte(tt.url))
				require.NoError(t, err, "url: %v", tt.url)

				r, err := http.NewRequest("POST", ts.URL, buf)
				require.NoError(t, err, "url: %v", tt.url)
				r.Header.Set("Content-Type", ctText)
				r.Header.Set("Content-Encoding", "gzip")

				res, err := cl.Do(r)
				require.NoError(t, err, "url: %v", tt.url)

				if res.StatusCode != http.StatusCreated {
					t.Errorf("failed when filling test data\nurl: %v\nstatus:%v", tt.url, res.StatusCode)

					return
				}

				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				require.NoError(t, res.Body.Close())
				assert.Equal(t, tt.want, string(body))
			})*/
	}
}

func Test_MakeShortJSON(t *testing.T) {
	cases := getTests()
	ts := newTestSrv()
	cl := newTestClient(ts)

	defer ts.Close()
	defer resetStorage()
	for _, tt := range cases {
		t.Run("POST JSON", func(t *testing.T) {
			require.NoError(t, resetStorage(), "failed to reset storage")

			req := urlreq{tt.url}
			w := bytes.NewBuffer(nil)
			enc := json.NewEncoder(w)
			require.NoError(t, enc.Encode(req), "url: %v", tt.url)

			res, err := cl.Post(ts.URL+"/api/shorten", ctJSON, w)
			require.NoError(t, err, "url: %v", tt.url)

			if res.StatusCode != http.StatusCreated {
				t.Errorf("failed when filling test data\nurl: %v\nstatus:%v", tt.url, res.StatusCode)

				return
			}

			message := urlres{}
			dec := json.NewDecoder(res.Body)
			err = dec.Decode(&message)
			if err != nil {
				message, _ := io.ReadAll(res.Body)
				require.NoError(t, err, "failed to parse message:", string(message))
			}

			require.NoError(t, res.Body.Close())
			assert.Equal(t, tt.want, message.Result, "result mismatch\nurl: %v\nstatus:%v", tt.url, res.StatusCode)
		})

		/*t.Run("POST JSON  gzip", func(t *testing.T) {
			require.NoError(t, resetStorage(), "failed to reset storage")

			r, w := io.Pipe()
			reqJSON := urlreq{tt.url}
			writer, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			require.NoError(t, err, "url: %v", tt.url)
			enc := json.NewEncoder(writer)
			require.NoError(t, enc.Encode(reqJSON), "url: %v", tt.url)
			require.NoError(t, writer.Close(), "url: %v", tt.url)

			req, err := http.NewRequest("POST", ts.URL+"/api/shorten", r)
			require.NoError(t, err, "url: %v", tt.url)
			req.Header.Set("Content-Type", ctJSON)
			req.Header.Set("Content-Encoding", "gzip")

			res, err := cl.Do(req)
			require.NoError(t, err, "url: %v", tt.url)

			if res.StatusCode != http.StatusCreated {
				body, err := io.ReadAll(res.Body)
				assert.NoError(t, err)
				t.Errorf("failed when filling test data\nurl: %v\nstatus:%v\nerr:%v", tt.url, res.StatusCode, string(body))

				return
			}

			message := urlres{}
			dec := json.NewDecoder(res.Body)
			err = dec.Decode(&message)
			if err != nil {
				message, _ := io.ReadAll(res.Body)
				require.NoError(t, err, "failed to parse message:", string(message))
			}

			require.NoError(t, res.Body.Close())
			assert.Equal(t, tt.want, message.Result, "result mismatch\nurl: %v\nstatus:%v", tt.url, res.StatusCode)
		})*/
	}

	t.Run("Wrong content type header", func(t *testing.T) {
		defer resetStorage()
		tt := cases[0]
		req := urlreq{tt.url}
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
	cases := getTests()
	ts := newTestSrv()

	defer ts.Close()

	t.Run("Empty storage", func(t *testing.T) {
		cl := newTestClient(ts)
		defer resetStorage()

		tt := cases[0]
		res, err := cl.Get(tt.want)
		require.NoError(t, err, "url: %v", tt.url)
		require.NoError(t, res.Body.Close())
		require.Equal(t, http.StatusNotFound, res.StatusCode, "got wrong status code\nurl:%v", tt.url)
	})
}

func Test_MakeLong(t *testing.T) {
	cases := getTests()
	ts := newTestSrv()
	cl := newTestClient(ts)

	defer ts.Close()
	t.Run("Get URL", func(t *testing.T) {
		defer resetStorage()
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
		if err := resetStorage(); err != nil {
			require.NoError(t, err, "failed to reset storage")
		}
		defer resetStorage()
		tt := cases[0]
		res, err := cl.Get(tt.want + "i")
		require.NoError(t, err, "url: %v", tt.url)
		require.NoError(t, res.Body.Close())
		assert.Equal(t, http.StatusNotFound, res.StatusCode, "got wrong status code\nurl%v:", tt.url)
	})

	tt := cases[0]
	t.Run("Get gzipMW", func(t *testing.T) {
		defer resetStorage()
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

func resetStorage() error {
	path := configrw.ReadStoragePath()

	// path is not set, quit wo error
	if path == "" {
		return nil
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// path does not exist, nothing to delete
		return nil
	}

	if err := os.Remove(path); err != nil {
		return err
	}
	fmt.Println("storage reset successful")

	return nil
}