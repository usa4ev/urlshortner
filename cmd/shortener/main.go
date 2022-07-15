package main

import (
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"

	shortner "github.com/usa4ev/urlshortner/internal/app"
)

func main() {
	r := chi.NewRouter()
	r.Route("/", chiRouter)
	log.Fatal(http.ListenAndServe("localhost:8080", r))
}

func chiRouter(r chi.Router) {
	storage := shortner.NewStorage()
	r.Post("/", makeShort(storage))
	r.Get("/{id}", makeLong(storage))
}

func makeShort(storage *shortner.Storage) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		URL, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		id := shortner.ShortURL(string(URL), storage)
		w.WriteHeader(http.StatusCreated)
		_, err = io.WriteString(w, "http://"+r.Host+"/"+strconv.Itoa(id))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func makeLong(storage *shortner.Storage) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		strid := r.URL.Path[1:]
		if strid == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(strid)
		if err != nil {
			http.Error(w, "id must be an integer", http.StatusBadRequest)
			return
		}

		redirect := shortner.GetPath(id, storage)
		if redirect == "" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
	}
}
