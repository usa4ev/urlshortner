package main

import (
	"log"
	"net/http"

	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/shortener"
)

func main() {
	configrw.ParseFlags()
	r := shortener.NewRouter()
	r.Route("/", shortener.DefaultRoute())
	log.Fatal(http.ListenAndServe(configrw.ReadSrvAddr(), r))
}
