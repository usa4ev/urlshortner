package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	host   string = "localhost:8080"
	ctJSON        = "application/json"
	ctXML         = "application/xml"
)

type args struct {
	id string
}
type want struct {
	response string
	code     int
}
type data []struct {
	id  string
	url string
}

func Test_jsonHandler(t *testing.T) {

	d := testData()
	ts := newTestSrv()
	cl := newTestClient(ts)

	defer ts.Close()

	type urlres struct {
		Result string `json:"result"`
	}

	type urlreq struct {
		URL string `json:"url"`
	}

	type urlreqx struct {
		URL string `json:"error"`
	}

	req := urlreq{}
	req.URL = d[0].url
	buf := bytes.NewBuffer([]byte(""))
	enc := json.NewEncoder(buf)
	err := enc.Encode(req)
	require.NoError(t, err)

	res, err := cl.Post(ts.URL+"/app/shorten", ctJSON, buf)

	require.NoError(t, err)
	defer res.Body.Close()
	for _, s := range d {
		req := urlreq{}
		req.URL = s.url
		buf := bytes.NewBuffer([]byte(""))
		enc := json.NewEncoder(buf)
		err := enc.Encode(req)
		require.NoError(t, err)

		res, err := cl.Post(ts.URL+"/app/shorten", ctJSON, buf)
		require.NoError(t, err)

		dec := json.NewDecoder(res.Body)

		if h := res.Header.Get("Content-Type"); h != ctJSON {
			t.Errorf("unexpected header. wanted %v, got %v", ctJSON, h)
			errinfo, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			t.Errorf("error text: %v", string(errinfo))
		}

		assert.Equal(t, http.StatusCreated, res.StatusCode)

		response := urlres{}
		err = dec.Decode(&response)
		require.NoError(t, err)

		expectedRes := ts.URL + "/" + s.id
		if response.Result != expectedRes {
			t.Errorf("failed when filling test data. Expecded url %v, got %v", expectedRes, response.Result)
		}
	}

	for _, s := range d {
		resp := urlreqx{}
		resp.URL = s.url
		buf := bytes.NewBuffer([]byte(""))
		enc := json.NewEncoder(buf)
		err := enc.Encode(resp)
		require.NoError(t, err)

		res, err := cl.Post(ts.URL+"/app/shorten", ctXML, buf)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	}
}

func Test_rootHandler(t *testing.T) {

	tests := getTests()
	d := testData()
	ts := newTestSrv()
	cl := newTestClient(ts)

	//check if it works fine with empty db

	defer ts.Close()

	res, err := cl.Get(ts.URL + "/1")
	require.NoError(t, err)

	err = res.Body.Close()
	require.NoError(t, err)

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("failed before data added, returned %v", res.StatusCode)
	}

	//filling test data
	for _, s := range d {
		res, err := cl.Post(ts.URL, "text/plain", bytes.NewBuffer([]byte(s.url)))
		require.NoError(t, err)
		if res.StatusCode != http.StatusCreated {
			t.Errorf("failed when filling test data, returned %v", res.StatusCode)
			continue
		}
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		err = res.Body.Close()
		require.NoError(t, err)
		expectedRes := ts.URL + "/" + s.id
		if string(body) != expectedRes {
			t.Errorf("failed when filling test data. Expecded id %v, got %v", expectedRes, string(body))
		}
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			res, err := cl.Get(ts.URL + "/" + tt.args.id)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, res.StatusCode)
			if tt.want.response != "" {

				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				err = res.Body.Close()
				require.NoError(t, err)
				assert.Equal(t, tt.want.response, string(body))
			}
		})
	}
}

func newTestClient(ts *httptest.Server) *http.Client {
	cl := ts.Client()

	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return cl
}

func newTestSrv() *httptest.Server {
	r := chi.NewRouter()
	r.Route("/", chiRouter)

	l, err := net.Listen("tcp", host)
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen on %v: %v", "localhost:8080", err))
	}

	ts := httptest.NewUnstartedServer(r)
	ts.Listener = l
	ts.Start()
	return ts
}

func testData() data {
	d := data{{"1", host + "/test"},
		{"2", host + "/test2"}}
	return d
}

func getTests() []struct {
	name string
	args args
	want want
} {
	tests := []struct {
		name string
		args args
		want want
	}{
		{"wrong id",
			args{id: "5"},
			want{"",
				404},
		},
		{"ok",
			args{id: "1"},
			want{"",
				307},
		},
		{"id is not int",
			args{id: "b"},
			want{"id must be an integer\n",
				400},
		},
	}
	return tests
}
