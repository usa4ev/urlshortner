package main

import (
	"github.com/usa4ev/urlshortner/internal/router"
	"log"
)

func main() {
	log.Fatal(router.ListenAndServe())
}
