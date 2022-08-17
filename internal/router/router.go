package router

import (
	"github.com/go-chi/chi"
	"net/http"
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
	r.Route("/", DefaultRoute(h))

	return r
}

func DefaultRoute(h handled) func(r chi.Router) {
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

func Route() func(r chi.Router) {
	/*return func(r chi.Router) {
		r.Get("/", Test)
	}*/
	return func(r chi.Router) {
		r.Method("GET", "/", http.HandlerFunc(Router{}.Test))
	}
}

func (g Router) Test(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusGone)
}
