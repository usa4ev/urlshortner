package router

import (
	"github.com/go-chi/chi"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"net/http"
)

type Router chi.Router

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
