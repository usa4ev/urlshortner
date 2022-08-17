package router

import (
	"net/http"

	"github.com/go-chi/chi"
)

type (
	Router  struct{ chi.Router }
	handled interface {
		Handlers() []HandlerDesc
	}
	HandlerDesc struct {
		Method      string
		Path        string
		Handler     http.Handler
		Middlewares chi.Middlewares
	}
)

func NewRouter(h handled) http.Handler {
	r := chi.NewRouter()
	r.Route("/", defaultRoute(h))

	return r
}

func defaultRoute(h handled) func(r chi.Router) {
	return func(r chi.Router) {
		for _, route := range h.Handlers() {
			r.With(route.Middlewares...).Method(route.Method, route.Path, route.Handler)
		}
	}
}

func Middlewares(h ...func(http.Handler) http.Handler) chi.Middlewares {
	mws := chi.Middlewares{}
	for _, f := range h {
		mws = append(mws, f)
	}

	return mws
}
