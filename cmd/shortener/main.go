package main

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/shortener"
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
	myShortener := shortener.NewShortener()
	//r := router.NewRouter(myShortener)
	r := router.NewRouter(myShortener)
	server := &http.Server{Addr: myShortener.Config.SrvAddr(), Handler: r}

	fmt.Println("addr: " + myShortener.Config.SrvAddr())
	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig

		// Trigger graceful shutdown
		myShortener.FlushStorage()
		server.Close()
	}()

	// Run the server
	err := server.ListenAndServe()

	fmt.Printf("listen and serve err: %v\n", err)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err.Error())
	}
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
