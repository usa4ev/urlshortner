package router

import (
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi"
	"github.com/usa4ev/urlshortner/internal/shortener"
)

type Router chi.Router

func NewRouter(s *shortener.MyShortener) Router {
	r := chi.NewRouter()

	r.Route("/", defaultRoute(s))

	return r
}

func ListenAndServe() error {
	myShortener := shortener.NewShortener()

	r := NewRouter(myShortener)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		myShortener.FlushStorage()
		close(c)
		return

	}()
	return http.ListenAndServe(myShortener.Config.SrvAddr(), r)
}

func defaultRoute(shortener *shortener.MyShortener) func(r chi.Router) {
	return func(r chi.Router) {
		for _, route := range shortener.Handlers() {
			r.With(route.Middlewares...).Method(route.Method, route.Path, route.Handler)
		}
	}
}
