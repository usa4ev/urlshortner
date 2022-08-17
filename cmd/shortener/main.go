package main

import (
	"context"
	"fmt"
	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// The HTTP Server
	fmt.Println("starting server..")
	config := configrw.NewConfig()
	myShortener := shortener.NewShortener()
	server := &http.Server{Addr: config.SrvAddr(), Handler: router.NewRouter(myShortener)}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	fmt.Println("addr: " + config.SrvAddr())
	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 30 seconds
		shutdownCtx, cancelCtx := context.WithTimeout(serverCtx, 30*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				defer cancelCtx()
				fmt.Println("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		myShortener.FlushStorage()
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			fmt.Println(err.Error())
		}
		serverStopCtx()
	}()

	// Run the server
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Println(err.Error())
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}
