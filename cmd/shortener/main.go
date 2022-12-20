package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"github.com/usa4ev/urlshortner/internal/storage"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func printMetaInfo() {
	fmt.Printf("Build version: %v\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %v\n", buildCommit)
}

func main() {
	printMetaInfo()

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
	if cfg.UseTLS() {
		err = server.ListenAndServeTLS(filepath.Join(cfg.SslPath(), "example.crt"), filepath.Join(cfg.SslPath(), "example.key"))
	} else {
		err = server.ListenAndServe()
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err.Error())
	}
}
