package main

import (
	"github.com/go-chi/chi"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"net/http"
)

func main() {
	router.ListenAndServe()
	//// The HTTP Server
	//myShortener := shortener.NewShortener()
	////r := router.NewRouter(myShortener)
	//r := NewRouter(myShortener)
	//server := &http.Server{Addr: "localhost:8080", Handler: r}
	//
	//// Server run context
	//serverCtx, serverStopCtx := context.WithCancel(context.Background())
	//fmt.Println("addr: " + myShortener.Config.SrvAddr())
	//// Listen for syscall signals for process to interrupt/quit
	//sig := make(chan os.Signal, 1)
	//signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	//
	//go func() {
	//	<-sig
	//
	//	// Shutdown signal with grace period of 30 seconds
	//	shutdownCtx, cancelCtx := context.WithTimeout(serverCtx, 30*time.Second)
	//
	//	go func() {
	//		<-shutdownCtx.Done()
	//		if shutdownCtx.Err() == context.DeadlineExceeded {
	//			defer cancelCtx()
	//			fmt.Println("graceful shutdown timed out.. forcing exit.")
	//		}
	//	}()
	//
	//	// Trigger graceful shutdown
	//	myShortener.FlushStorage()
	//	err := server.Shutdown(shutdownCtx)
	//	if err != nil {
	//		fmt.Println(err.Error())
	//	}
	//	serverStopCtx()
	//}()
	//
	//// Run the server
	//err := server.ListenAndServe()
	//if err != nil && !errors.Is(err, http.ErrServerClosed) {
	//	panic(err.Error())
	//}
	//
	//// Wait for server context to be stopped
	//<-serverCtx.Done()
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
