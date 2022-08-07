package router

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Router chi.Router

func ListenAndServe() {
	// The HTTP Server
	config := configrw.NewConfig()
	myShortener := shortener.NewShortener()
	server := &http.Server{Addr: config.SrvAddr(), Handler: NewRouter(myShortener)}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, _ := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		myShortener.FlushStorage()
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	// Run the server
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}

func NewRouter(myShortener *shortener.MyShortener) http.Handler {
	r := chi.NewRouter()
	r.Route("/", defaultRoute(myShortener))

	return r
}

func defaultRoute(shortener *shortener.MyShortener) func(r chi.Router) {
	return func(r chi.Router) {
		for _, route := range shortener.Handlers() {
			r.With(route.Middlewares...).Method(route.Method, route.Path, route.Handler)
		}
	}
}
