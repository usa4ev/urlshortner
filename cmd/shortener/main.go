package main

import (
	shortner "github.com/usa4ev/urlshortner/internal/app"
	"io"
	"log"
	"net/http"
	"strconv"
)

func main() {

	listen()

}

func listen() {
	http.HandleFunc("/", rootHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:

		URL, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		id := shortner.ShortURL(string(URL))

		_, err = io.WriteString(w, strconv.Itoa(id))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodGet:
		id, err := strconv.Atoi(r.URL.Path[1:])
		if err != nil {
			http.Error(w, "id must be an integer", http.StatusBadRequest)
			return
		}

		redirect := shortner.GetPath(id)
		if redirect == "" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, redirect, http.StatusMovedPermanently)
	}
}
