package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	conf "github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/shortener"
	"github.com/usa4ev/urlshortner/internal/storage"
)

const (
	addr = "localhost:8080"
	url  = "http://localhost"
)

func Example() {
	vars := map[string]string{
		"SERVER_ADDRESS": addr,
		"BASE_URL":       url}

	cfg := conf.New(conf.IgnoreOsArgs(),
		conf.WithEnvVars(vars))

	strg, _ := storage.New(cfg)

	myShortner := shortener.NewShortener(cfg, strg)

	server := New(cfg, myShortner, strg)

	server.Run()
	defer server.Shutdown(context.Background())

	// Getting a short URL
	res, _ := http.Post(url+"/",
		"text/plain",
		bytes.NewBuffer([]byte("example.com")))

	shortURL, _ := io.ReadAll(res.Body)

	res.Body.Close()

	// Getting a short URL using JSON-encoded message
	message := struct {
		URL string `json:"url"`
	}{"exampleJSON.com"}
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.Encode(message)

	res, _ = http.Post(url+"/api/shorten",
		"application/json",
		buf)

	shortURLjson, _ := io.ReadAll(res.Body)

	res.Body.Close()

	// accessing the original address
	res, _ = http.Get(string(shortURL))

	res.Body.Close()

	res, _ = http.Get(string(shortURLjson))

	res.Body.Close()
	// service redirects request to the original URL
}
