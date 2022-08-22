package main

import (
	"errors"
	"fmt"
	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"github.com/usa4ev/urlshortner/internal/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("srv start...")
	defer fmt.Println("srv exit")

	fmt.Printf("vars: %v", os.Environ())

	os.Environ()
	// The HTTP Server
	cfg := config.New()
	strg := storage.New(cfg)
	myShortener := shortener.NewShortener(cfg, strg)
	r := router.NewRouter(myShortener)
	server := &http.Server{Addr: cfg.SrvAddr(), Handler: r}

	fmt.Println("addr: " + cfg.SrvAddr())
	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig

		// Trigger graceful shutdown
		strg.Flush()
		server.Close()
	}()

	// Run the server
	err := server.ListenAndServe()

	fmt.Printf("listen and serve err: %v\n", err)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err.Error())
	}
}
