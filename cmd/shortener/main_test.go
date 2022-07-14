package main

import (
	"bytes"
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

var host string

func Test_rootHandler(t *testing.T) {
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
	d := data{{"1", host + "/test"},
		{"2", host + "/test2"}}

	//check if it works fine with empty db

	r := chi.NewRouter()
	r.Route("/", chiRouter)

	l, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		panic(fmt.Sprintf("httptest: failed to listen on %v: %v", "localhost:8080", err))
	}

	ts := httptest.NewUnstartedServer(r)
	ts.Listener = l
	ts.Start()

	cl := ts.Client()
	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	host = ts.URL
	defer ts.Close()
	res, err := cl.Get(host + "/1")
	require.NoError(t, err)
	err = res.Body.Close()
	require.NoError(t, err)

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("failed before data added, returned %v", res.StatusCode)
	}

	//filling test data
	for _, s := range d {
		res, err := cl.Post(host, "text/plain", bytes.NewBuffer([]byte(s.url)))
		require.NoError(t, err)
		if res.StatusCode != http.StatusCreated {
			t.Errorf("failed when filling test data, returned %v", res.StatusCode)
			continue
		}
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		err = res.Body.Close()
		require.NoError(t, err)
		expectedRes := host + "/" + s.id
		if string(body) != expectedRes {
			t.Errorf("failed when filling test data. Expecded id %v, got %v", expectedRes, string(body))
		}
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			res, err := cl.Get(host + "/" + tt.args.id)
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
