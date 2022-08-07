package router

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/usa4ev/urlshortner/internal/shortener"
)

type Router chi.Router

func NewRouter() Router {
	r := chi.NewRouter()
	myShortener := shortener.NewShortener()

	r.Route("/", defaultRoute(*myShortener))

	return r
}

func ListenAndServe(r Router) error {
	myShortener := shortener.NewShortener()

	defer myShortener.FlushStorage()

	return http.ListenAndServe(myShortener.Config.SrvAddr(), r)
}

func defaultRoute(shortener shortener.MyShortener) func(r chi.Router) {
	return func(r chi.Router) {
		for _, route := range shortener.Handlers() {
			r.With(route.Middlewares...).Method(route.Method, route.Path, route.Handler)
		}
	}
}
