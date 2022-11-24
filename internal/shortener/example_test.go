package shortener

import (
	"bytes"
	"encoding/json"
	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/storage"
	"io"
	"net/http"
)

const (
	addr = "localhost:8080"
	url  = "http://localhost"
)

func Example() {
	vars := map[string]string{
		"SERVER_ADDRESS": addr,
		"BASE_URL":       url}

	cfg := config.New(config.IgnoreOsArgs(),
		config.WithEnvVars(vars))

	strg, _ := storage.New(cfg)

	myShortner := NewShortener(cfg, strg)

	r := router.NewRouter(myShortner)

	server := &http.Server{Addr: cfg.SrvAddr(), Handler: r}
	server.ListenAndServe()

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
