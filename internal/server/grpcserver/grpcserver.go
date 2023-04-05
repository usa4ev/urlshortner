// Package grpcserver is a grpc implementation of shortener server.
package grpcserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/usa4ev/urlshortner/internal/server/auth"
	ps "github.com/usa4ev/urlshortner/internal/server/grpcserver/protoshortener"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"github.com/usa4ev/urlshortner/internal/storage/database"
	"github.com/usa4ev/urlshortner/internal/storage/storageerrors"
)

type config interface {
	UseTLS() bool
	SslPath() string
	DBDSN() string
	SrvAddr() string
	TrustedSubnet() string
}

type Server struct {
	ps.ShortenerServer

	shortener  shortener.Shortener
	sessionMgr auth.SessionStoreLoader
	cfg        config
	sfgr       *singleflight.Group
	gs         *grpc.Server
}

func New(c config, s shortener.Shortener, sm auth.SessionStoreLoader) *Server {
	srv := Server{}

	srv.cfg = c
	srv.shortener = s
	srv.sessionMgr = sm
	srv.sfgr = new(singleflight.Group)

	return &srv
}

func (srv *Server) listenAndServe() error {
	listen, err := net.Listen("tcp", srv.cfg.SrvAddr())
	if err != nil {
		return err
	}

	interceptor := grpc.UnaryServerInterceptor(srv.authInterceptor)

	gs := grpc.NewServer(grpc.Creds(insecure.NewCredentials()),
		grpc.UnaryInterceptor(interceptor))
	srv.gs = gs
	ps.RegisterShortenerServer(gs, srv)
	fmt.Println("gRPC server starts")

	if err := gs.Serve(listen); err != nil {
		return err
	}

	return nil
}

func (srv *Server) listenAndServeTLS(cert, key string) error {
	interceptor := grpc.UnaryServerInterceptor(srv.authInterceptor)

	crt, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return err
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{crt},
		ClientAuth:   tls.NoClientCert,
	})

	gs := grpc.NewServer(grpc.Creds(creds),
		grpc.UnaryInterceptor(interceptor))

	ps.RegisterShortenerServer(gs, srv)
	fmt.Println("gRPC server starts")

	listen, err := net.Listen("tcp", srv.cfg.SrvAddr())
	if err != nil {
		return err
	}

	if err := gs.Serve(listen); err != nil {
		return err
	}

	return nil
}

func (srv *Server) Run() error {
	// Run the server
	if srv.cfg.UseTLS() {
		return srv.listenAndServeTLS(
			filepath.Join(srv.cfg.SslPath(), "example.crt"),
			filepath.Join(srv.cfg.SslPath(), "example.key"))
	} else {
		return srv.listenAndServe()
	}
}

func (srv *Server) Shutdown(ctx context.Context) error {
	if err := srv.shortener.FlushStorage(); err != nil {
		return err
	}

	if srv.gs != nil {
		srv.gs.GracefulStop()
	}

	return nil
}

func (srv *Server) authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	var token, userID string

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("authorization")
		if len(values) > 0 {
			token = values[0]
		}
	}

	if token != "" {
		//token is set, look up the user
		userID, err = auth.LoadUser(token, srv.sessionMgr)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "failed to load user by token")
		}
	} else {
		//token is not set, open new session
		userID, _, err = auth.OpenSession(srv.sessionMgr)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "failed to open new session")
		}
	}

	md, _ := metadata.FromIncomingContext(ctx)
	md.Set("user_id", userID)

	ctx = metadata.NewIncomingContext(ctx, md)

	return handler(ctx, req)
}

func (srv *Server) Shorten(ctx context.Context, in *ps.ShortenRequest) (*ps.ShortenResponse, error) {
	res := ps.ShortenResponse{}

	userID, err := getUserID(ctx)
	if err != nil {
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	}

	id, _ := srv.shortener.ShortenURL(in.Url)

	err = srv.shortener.StoreURL(id, in.Url, userID)
	if err != nil {
		if errors.Is(err, storageerrors.ErrConflict) {
			res.Error = err.Error()
			return &res, status.Errorf(codes.AlreadyExists, "url %v is already shortened. id: %v", in.Url, id)
		}

		return &res, status.Errorf(codes.Internal, "failed to store URL: %v", err.Error())
	}

	res.Id = id

	return &res, nil
}

func getUserID(ctx context.Context) (string, error) {
	val := metadata.ValueFromIncomingContext(ctx, "user_id")
	if len(val) != 1 || val[0] == "" {
		return "", fmt.Errorf("user ID is not defined")
	}
	return val[0], nil
}

func (srv *Server) ShortenBatch(ctx context.Context, in *ps.ShortenBatchRequest) (*ps.ShortenBatchResponse, error) {
	res := ps.ShortenBatchResponse{}
	data := make([]*ps.URLwId, 0)

	userID, err := getUserID(ctx)
	if err != nil {
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	}

	for _, v := range in.Data {
		id, url := srv.shortener.ShortenURL(v.Url)
		data = append(data, &ps.URLwId{Id: v.Id, Url: url})
		err = srv.shortener.StoreURL(id, v.Url, userID)
		if err != nil {
			res.Error = err.Error()
			return &res, status.Errorf(codes.Internal, "failed to store URL: %v", err.Error())
		}
	}

	res.Data = data

	return &res, nil
}

func (srv *Server) GetLong(ctx context.Context, in *ps.GetLongRequest) (*ps.GetLongResponse, error) {
	res := ps.GetLongResponse{}
	redirect, err := srv.shortener.FindURL(in.Id)
	switch {
	case errors.Is(err, storageerrors.ErrURLGone):
		res.Error = err.Error()
		return &res, status.Errorf(codes.Unavailable, "URL deleted: %v", err.Error())
	case err != nil && !errors.Is(err, storageerrors.ErrURLGone):
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	case redirect == "":
		res.Error = err.Error()
		return &res, status.Errorf(codes.NotFound, "URL not found; id: %v", in.Id)
	}

	res.Url = redirect

	return &res, nil
}

func (srv *Server) GetLongByUser(ctx context.Context, in *ps.Dummy) (*ps.GetLongByUserResponse, error) {
	res := ps.GetLongByUserResponse{}
	data := make([]string, 0)

	userID, err := getUserID(ctx)
	if err != nil {
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	}

	pairs, err := srv.shortener.LoadByUser(userID)
	if err != nil {
		res.Error = err.Error()
		return &res, status.Errorf(codes.Internal, "failed to load URLs by user: %v", err.Error())
	}

	if len(pairs) == 0 {
		return &res, status.Errorf(codes.NotFound, "no URLs found by user: %v", userID)
	}

	for _, v := range pairs {
		data = append(data, v.ShortURL)
	}

	res.Urls = data

	return &res, nil
}

func (srv *Server) DeleteBatch(ctx context.Context, in *ps.DeleteBatchRequest) (*ps.DeleteBatchResponse, error) {
	res := ps.DeleteBatchResponse{}
	userID, err := getUserID(ctx)
	if err != nil {
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	}

	err = srv.shortener.DeleteURLs(userID, in.Ids)

	if err != nil {
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	}

	return &res, nil
}

func (srv *Server) Stats(ctx context.Context, in *ps.Dummy) (*ps.StatsResponse, error) {
	res := ps.StatsResponse{}

	pr, ok := peer.FromContext(ctx)
	if !ok {
		res.Error = "failed to get peer from ctx"
		return &res, status.Error(codes.Unauthenticated, res.Error)

	}
	if pr.Addr == net.Addr(nil) {
		res.Error = "failed to get peer address"
		return &res, status.Error(codes.Unauthenticated, res.Error)

	}

	//parse subnet string
	_, subnet, err := net.ParseCIDR(srv.cfg.TrustedSubnet())
	if err != nil {
		res.Error = fmt.Sprintf("failed to parse trusted subnet from config: %v", srv.cfg.TrustedSubnet())
		return &res, status.Error(codes.Internal, res.Error)

	}

	ip := net.ParseIP(pr.Addr.String())

	//check if ip belongs subnet
	if !subnet.Contains(ip) {
		res.Error = "call from unathorized subnet"
		return &res, status.Error(codes.Unauthenticated, res.Error)
	}

	//use SingleFlight
	urls, err, _ := srv.sfgr.Do("CountURLs",
		func() (interface{}, error) {
			return srv.shortener.CountURLs()
		})
	if err != nil {
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	}

	users, err, _ := srv.sfgr.Do("CountUsers",
		func() (interface{}, error) {
			return srv.shortener.CountUsers()
		})

	if err != nil {
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	}

	res.Users = users.(int32)
	res.Urls = urls.(int32)

	return &res, nil
}

func (srv *Server) PingStorage(ctx context.Context, in *ps.Dummy) (*ps.PingStorageResponse, error) {
	res := ps.PingStorageResponse{}

	err := database.Pingdb(srv.cfg.DBDSN())
	if err != nil {
		res.Error = err.Error()
		return &res, status.Error(codes.Internal, err.Error())
	}

	return &res, nil
}
