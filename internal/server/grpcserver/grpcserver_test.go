package grpcserver

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	conf "github.com/usa4ev/urlshortner/internal/config"
	ps "github.com/usa4ev/urlshortner/internal/server/grpcserver/protoshortener"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"github.com/usa4ev/urlshortner/internal/storage"
)

type (
	test struct {
		url  string
		want string
		id   string
	}
	tests []test
)

func getTests(baseURL string) tests {
	urls := []string{
		"http://ya.ru/test",
		"http://vk.com/test/test2",
		"https://xn--80aesfpebagmfblc0a.xn--p1ai/test/test2/",
	}
	res := tests{}

	var id string
	for _, v := range urls {
		id = base64.RawURLEncoding.EncodeToString([]byte(v))
		res = append(res, test{v, baseURL + "/" + id, id})
	}

	return res
}

func newTestClient(cfg *conf.Config) ps.ShortenerClient {
	conn, err := grpc.Dial(cfg.SrvAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	return ps.NewShortenerClient(conn)
}

func newTestSrv(cfg *conf.Config) (*Server, error) {
	strg, err := storage.New(cfg)
	if err != nil {
		return nil, err
	}

	s := shortener.NewShortener(cfg, strg)

	srv := Server{}

	srv.cfg = cfg
	srv.shortener = s
	srv.sessionMgr = strg
	srv.sfgr = new(singleflight.Group)

	ts := New(cfg, s, strg)

	go func() { log.Fatal(ts.ListenAndServe()) }()

	return ts, nil
}

func TestServer_Shorten(t *testing.T) {
	cfg := testcfg()

	cases := getTests(cfg.BaseURL())
	resetStorage(cfg.StoragePath(), cfg.DBDSN())
	ts, err := newTestSrv(cfg)
	require.NoError(t, err)
	defer ts.Shutdown(context.Background())

	cl := newTestClient(cfg)

	ctx := context.Background()

	for _, tt := range cases {
		t.Run("shorten", func(t *testing.T) {
			in := &ps.ShortenRequest{Url: tt.url}
			out, err := cl.Shorten(ctx, in)
			require.NoError(t, err, "url: %v", tt.url)

			assert.Equal(t, tt.id, out.Id)
		})
	}

	for _, tt := range cases {
		t.Run("shorten with conflict", func(t *testing.T) {
			in := &ps.ShortenRequest{Url: tt.url}
			_, err := cl.Shorten(ctx, in)
			if status.Code(err) != codes.AlreadyExists {
				t.Errorf("failed when there is conflict \nurl: %v\nstatus:%v", tt.url, status.Code(err))

				return
			}
		})
	}
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

	return nil
}

func testcfg() *conf.Config {
	return conf.New(conf.WithEnvVars(map[string]string{
		"BASE_URL":          "http://localhost:8080",
		"SERVER_ADDRESS":    "localhost:8080",
		"FILE_STORAGE_PATH": os.Getenv("HOME") + "/storage.csv",
	}),
		conf.IgnoreOsArgs())
}
