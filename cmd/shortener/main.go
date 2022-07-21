package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi"
	shortner "github.com/usa4ev/urlshortner/internal/app"
)

func main() {
	r := chi.NewRouter()
	r.Route("/", chiRouter)
	log.Fatal(http.ListenAndServe("localhost:8080", r))
}

func chiRouter(r chi.Router) {
	shortener := shortner.MyShortener{}
	r.Post("/", shortener.MakeShort)
	r.Get("/{id}", shortener.MakeLong)
	r.Post("/api/shorten", shortener.MakeShortJSON)
}
