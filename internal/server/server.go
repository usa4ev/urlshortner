package server

import (
	"context"

	"github.com/usa4ev/urlshortner/internal/server/auth"
	"github.com/usa4ev/urlshortner/internal/server/grpcserver"
	"github.com/usa4ev/urlshortner/internal/server/httpserver"
	"github.com/usa4ev/urlshortner/internal/shortener"
)

type (
	Server interface {
		ListenAndServe() error
		ListenAndServeTLS(cert, key string) error
		Shutdown(ctx context.Context) error
	}

	config interface {
		GRPC() bool
		UseTLS() bool
		SslPath() string
		DBDSN() string
		SrvAddr() string
		TrustedSubnet() string
	}
)

func New(c config, s shortener.Shortener, sm auth.SessionStoreLoader) Server {

	if c.GRPC() {
		return grpcserver.New(c, s, sm)
	} else {
		return httpserver.New(c, s, sm)
	}

}
