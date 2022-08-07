package shortener

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/storage"
)

const (
	ctJSON string = "application/json"
)

type (
	MyShortener struct {
		storage  *storage.Storage
		Config   configrw.Config
		handlers []handler
	}
	urlreq struct {
		URL string `json:"url"`
	}
	urlres struct {
		Result string `json:"result"`
	}
	handler struct {
		Method      string
		Path        string
		Handler     http.Handler
		Middlewares chi.Middlewares
	}
)

func NewShortener() *MyShortener {
	s := &MyShortener{}
	s.Config = configrw.NewConfig()
	s.storage = storage.NewStorage(s.Config.StoragePath())
	s.handlers = []handler{
		{"POST", "/", http.HandlerFunc(s.makeShort), chi.Middlewares{gzipMW}},
		{"GET", "/{id}", http.HandlerFunc(s.makeLong), chi.Middlewares{gzipMW}},
		{"POST", "/api/shorten", http.HandlerFunc(s.makeShortJSON), chi.Middlewares{gzipMW}},
	}

	return s
}

func (myShortener *MyShortener) makeShort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	URL, err := readBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	shortURL := myShortener.shortenURL(string(URL))

	w.WriteHeader(http.StatusCreated)

	_, err = io.WriteString(w, shortURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *MyShortener) makeShortJSON(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != ctJSON {
		http.Error(w, "unsupported content type", http.StatusBadRequest)

		return
	}

	defer r.Body.Close()

	body, err := readBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	message := urlreq{}
	dec := json.NewDecoder(bytes.NewBuffer(body))

	if err := dec.Decode(&message); err != nil {
		http.Error(w, "failed to decode message: "+err.Error(), http.StatusBadRequest)

		return
	}

	res := urlres{myShortener.shortenURL(message.URL)}

	w.Header().Set("Content-Type", ctJSON)
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)

	if err := enc.Encode(res); err != nil {
		http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *MyShortener) shortenURL(url string) string {
	key := base64.RawURLEncoding.EncodeToString([]byte(url))
	storagePath := myShortener.Config.StoragePath()

	if err := myShortener.storage.Append(key, url, storagePath); err != nil {
		panic("failed to write new values to storage:" + err.Error())
	}

	return myShortener.makeURL(key)
}

func (myShortener *MyShortener) makeURL(key string) string {
	return myShortener.Config.BaseURL() + "/" + key
}

func (myShortener *MyShortener) makeLong(w http.ResponseWriter, r *http.Request) {
	redirect, err := findURL(r.URL.Path[1:], myShortener)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	}

	http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
}

func findURL(key string, myShortener *MyShortener) (string, error) {
	if url, ok := myShortener.storage.Load(key); ok {
		return url, nil
	}

	return "", errors.New("url not found")
}

/*func NewRouter() *chi.Mux {
	return chi.NewRouter()
}*/

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func readBody(r *http.Request) ([]byte, error) {
	var reader io.Reader

	if r.Header.Get(`Content-Encoding`) == `gzip` {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}

		reader = gz
		defer gz.Close()
	} else {
		reader = r.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func gzipMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		supportedEncoding := r.Header.Values("Accept-Encoding")
		if len(supportedEncoding) > 0 {
			for _, v := range supportedEncoding {
				if v == "gzip" {
					writer, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)

						return
					}
					defer writer.Close()
					w.Header().Set("Content-Encoding", "gzip")
					gzipW := gzipWriter{w, writer}
					next.ServeHTTP(gzipW, r)

					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (myShortener *MyShortener) Handlers() []handler {
	return myShortener.handlers
}

func (myShortener *MyShortener) FlushStorage() {
	if err := myShortener.storage.Flush(); err != nil {
		log.Fatal(err)
	}
}
