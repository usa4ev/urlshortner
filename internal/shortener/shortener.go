package shortener

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/storage"
)

const (
	ctJSON string = "application/json"
)

type (
	myShortener struct {
		urlMap  storage.Storage
		baseURL string
	}
	urlreq struct {
		URL string `json:"url"`
	}
	urlres struct {
		Result string `json:"result"`
	}
)

func newShortener() *myShortener {
	return &myShortener{storage.NewStorage(), configrw.ReadBaseURL()}
}

func (myShortener *myShortener) MakeShort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	URL, err := readBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	shortURL := shortenURL(string(URL), myShortener)

	w.WriteHeader(http.StatusCreated)

	_, err = io.WriteString(w, shortURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *myShortener) MakeShortJSON(w http.ResponseWriter, r *http.Request) {
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

	res := urlres{shortenURL(message.URL, myShortener)}

	w.Header().Set("Content-Type", ctJSON)
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)

	if err := enc.Encode(res); err != nil {
		http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

		return
	}
}

func shortenURL(url string, myShortener *myShortener) string {
	if myShortener.urlMap == nil {
		myShortener.urlMap = make(storage.Storage)
	}

	for k, v := range myShortener.urlMap {
		if v == url {
			return myShortener.makeURL(k)
		}
	}

	key := base64.RawURLEncoding.EncodeToString([]byte(url))
	myShortener.urlMap[key] = url

	err := storage.AppendStorage(key, url)
	if err != nil {
		panic("failed to write new values to storage:" + err.Error())
	}

	return myShortener.makeURL(key)
}

func (myShortener *myShortener) makeURL(key string) string {
	return myShortener.baseURL + "/" + key
}

func (myShortener *myShortener) MakeLong(w http.ResponseWriter, r *http.Request) {
	redirect, err := findURL(r.URL.Path[1:], myShortener)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	}

	http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
}

func findURL(key string, myShortener *myShortener) (string, error) {
	if url, ok := myShortener.urlMap[key]; ok {
		return url, nil
	}

	return "", errors.New("url not found")
}

func NewRouter() *chi.Mux {
	return chi.NewRouter()
}

func DefaultRoute() func(r chi.Router) {
	return func(r chi.Router) {
		shortener := newShortener()
		r.With(gzipMW).Post("/", shortener.MakeShort)
		r.With(gzipMW).Get("/{id}", shortener.MakeLong)
		r.With(gzipMW).Post("/api/shorten", shortener.MakeShortJSON)
	}
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
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
