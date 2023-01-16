package httpserver

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi"
	"golang.org/x/sync/singleflight"

	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/server/auth"
	"github.com/usa4ev/urlshortner/internal/server/httpserver/middleware"
	"github.com/usa4ev/urlshortner/internal/shortener"
)

const (
	ctJSON string = "application/json"
)

type config interface {
	TrustedSubnet() string
	DBDSN() string
	SslPath() string
	UseTLS() bool
	SrvAddr() string
}

type Server struct {
	httpsrv    *http.Server
	shortener  shortener.Shortener
	sessionMgr auth.SessionStoreLoader
	cfg        config
	sfgr       *singleflight.Group
	handlers   []router.HandlerDesc //list of handlers that serve HTTP methods
}

func New(c config, s shortener.Shortener, sm auth.SessionStoreLoader) *Server {
	srv := Server{}

	srv.cfg = c
	srv.shortener = s
	srv.sessionMgr = sm
	srv.sfgr = new(singleflight.Group)
	srv.handlers = []router.HandlerDesc{
		{Method: "POST", Path: "/", Handler: http.HandlerFunc(srv.makeShort), Middlewares: chi.Middlewares{middleware.GzipMW, middleware.AuthMW(sm)}},
		{Method: "GET", Path: "/{id}", Handler: http.HandlerFunc(srv.makeLong), Middlewares: chi.Middlewares{middleware.GzipMW, middleware.AuthMW(sm)}},
		{Method: "POST", Path: "/api/shorten", Handler: http.HandlerFunc(srv.makeShortJSON), Middlewares: chi.Middlewares{middleware.GzipMW, middleware.AuthMW(sm)}},
		{Method: "POST", Path: "/api/shorten/batch", Handler: http.HandlerFunc(srv.shortenBatchJSON), Middlewares: chi.Middlewares{middleware.GzipMW, middleware.AuthMW(sm)}},
		{Method: "GET", Path: "/api/user/urls", Handler: http.HandlerFunc(srv.makeLongByUser), Middlewares: chi.Middlewares{middleware.GzipMW, middleware.AuthMW(sm)}},
		{Method: "DELETE", Path: "/api/user/urls", Handler: http.HandlerFunc(srv.deleteBatch), Middlewares: chi.Middlewares{middleware.GzipMW, middleware.AuthMW(sm)}},
		{Method: "GET", Path: "/ping", Handler: http.HandlerFunc(srv.pingStorage), Middlewares: chi.Middlewares{middleware.GzipMW, middleware.AuthMW(sm)}},
		{Method: "GET", Path: "/api/internal/stats", Handler: http.HandlerFunc(srv.stats), Middlewares: chi.Middlewares{middleware.GzipMW}},
	}

	r := router.NewRouter(&srv)
	srv.httpsrv = &http.Server{Addr: c.SrvAddr(), Handler: r}

	return &srv
}

func (srv *Server) Run() error {
	// Run the server
	if srv.cfg.UseTLS() {
		return srv.httpsrv.ListenAndServeTLS(
			filepath.Join(srv.cfg.SslPath(), "example.crt"), 
			filepath.Join(srv.cfg.SslPath(), "example.key"))
	} else {
		return srv.httpsrv.ListenAndServe()
	}
}

func (srv *Server) Shutdown(ctx context.Context)error {
	srv.httpsrv.Shutdown(ctx)
	return nil
}

func (srv *Server) Handlers() []router.HandlerDesc {
	return srv.handlers
}
