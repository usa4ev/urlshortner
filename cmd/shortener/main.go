package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"github.com/usa4ev/urlshortner/internal/storage"
)

func main() {
	os.Environ()
	// The HTTP Server
	cfg := config.New()
	strg, err := storage.New(cfg)
	if err != nil {
		panic(err.Error())
	}

	myShortener := shortener.NewShortener(cfg, strg)
	r := router.NewRouter(myShortener)
	server := &http.Server{Addr: cfg.SrvAddr(), Handler: r}

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		call := <-sig

		// Trigger graceful shutdown
		strg.Flush()
		server.Close()

		fmt.Printf("graceful shutdown, got call: %v\n", call.String())
	}()

	// Run the server
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err.Error())
	}
}
