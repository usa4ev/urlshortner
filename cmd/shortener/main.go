package main

import (
	"log"

	"github.com/usa4ev/urlshortner/internal/router"
)

func main() {
	r := router.NewRouter()
	log.Fatal(router.ListenAndServe(r))
}
