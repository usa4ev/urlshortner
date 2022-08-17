package shortener

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/storage"
	"github.com/usa4ev/urlshortner/internal/storage/database"
)

const (
	ctJSON       string     = "application/json"
	key          string     = "9cc1ee455a3363ffc504f40006f70d0c8276648a5d3eb3f9524e94d1b7a83aef"
	ctxKeyUserID contextKey = 0
)

type (
	MyShortener struct {
		storage  *storage.Storage
		Config   configrw.Config
		Handlers []router.HandlerDesc
	}
	urlreq struct {
		URL string `json:"url"`
	}
	urlres struct {
		Result string `json:"result"`
	}
	urlwid struct {
		CorrelationID string `json:"correlation_id"`
		OriginalURL   string `json:"original_url"`
	}
	urlwidres struct {
		CorrelationID string `json:"correlation_id"`
		ShortURL      string `json:"short_url"`
	}
	contextKey int
)

func NewShortener() *MyShortener {
	s := &MyShortener{}
	s.Config = configrw.NewConfig()
	s.storage = storage.New(s.Config)
	s.Handlers = []router.HandlerDesc{
		{Method: "POST", Path: "/", Handler: http.HandlerFunc(s.makeShort), Middlewares: router.Middlewares(gzipMW, s.authMW)},
		{Method: "POST", Path: "/api/shorten", Handler: http.HandlerFunc(s.makeShortJSON), Middlewares: router.Middlewares(gzipMW, s.authMW)},
		{Method: "POST", Path: "/api/shorten/batch", Handler: http.HandlerFunc(s.shortenBatchJSON), Middlewares: router.Middlewares(gzipMW, s.authMW)},
		{Method: "GET", Path: "/{id}", Handler: http.HandlerFunc(s.makeLong), Middlewares: router.Middlewares(gzipMW, s.authMW)},
		{Method: "GET", Path: "/api/user/urls", Handler: http.HandlerFunc(s.makeLongByUser), Middlewares: router.Middlewares(gzipMW, s.authMW)},
		{Method: "GET", Path: "/ping", Handler: http.HandlerFunc(s.pingStorage), Middlewares: router.Middlewares(s.authMW)},
	}

	return s
}

func (myShortener *MyShortener) pingStorage(w http.ResponseWriter, r *http.Request) {
	err := storage.Ping(myShortener.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (myShortener *MyShortener) makeShort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	userID := r.Context().Value(ctxKeyUserID).(string)
	originalURL, err := readBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	id, url := myShortener.shortenURL(string(originalURL))
	err = myShortener.storeURL(id, string(originalURL), userID)
	if err != nil {
		if errors.Is(err, database.ErrConflict) {
			http.Error(w, url, http.StatusConflict)

			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)

	_, err = io.WriteString(w, url)
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

	var userID string
	if rawUserID := r.Context().Value(ctxKeyUserID); rawUserID != nil {
		userID = rawUserID.(string)
	}

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

	enc := json.NewEncoder(w)
	id, url := myShortener.shortenURL(message.URL)
	res := urlres{url}
	err = myShortener.storeURL(id, message.URL, userID)
	if err != nil {
		if errors.Is(err, database.ErrConflict) {
			errorText := err.Error()

			if err := enc.Encode(res); err != nil {
				http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

				return
			}

			http.Error(w, errorText, http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", ctJSON)
	w.WriteHeader(http.StatusCreated)

	if err := enc.Encode(res); err != nil {
		http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *MyShortener) shortenBatchJSON(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != ctJSON {
		http.Error(w, "unsupported content type", http.StatusBadRequest)

		return
	}

	defer r.Body.Close()

	var userID string
	if rawUserID := r.Context().Value(ctxKeyUserID); rawUserID != nil {
		userID = rawUserID.(string)
	}

	body, err := readBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	message := make([]urlwid, 0)
	res := make([]urlwidres, 0)
	dec := json.NewDecoder(bytes.NewBuffer(body))

	if err := dec.Decode(&message); err != nil {
		http.Error(w, "failed to decode message: "+err.Error(), http.StatusBadRequest)

		return
	}

	for _, v := range message {
		id, url := myShortener.shortenURL(v.OriginalURL)
		res = append(res, urlwidres{v.CorrelationID, url})
		err = myShortener.storeURL(id, url, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}
	}

	w.Header().Set("Content-Type", ctJSON)
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)

	if err := enc.Encode(res); err != nil {
		http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *MyShortener) shortenURL(url string) (string, string) {
	id := base64.RawURLEncoding.EncodeToString([]byte(url))
	return id, myShortener.makeURL(id)
}

func (myShortener *MyShortener) storeURL(id, url, userID string) error {
	return myShortener.storage.StoreURL(id, url, userID)
}

func (myShortener *MyShortener) makeURL(id string) string {
	return myShortener.Config.BaseURL() + "/" + id
}

func (myShortener *MyShortener) makeLong(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[1:]
	redirect, err := myShortener.findURL(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	} else if redirect == "" {
		http.Error(w, fmt.Sprintf("id %v not found", id), http.StatusNotFound)

		return
	}

	http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
}

func (myShortener *MyShortener) makeLongByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxKeyUserID)
	res, err := myShortener.storage.LoadByUser(myShortener.makeURL, userID.(string))
	if err != nil {
		http.Error(w, "failed to load data: "+err.Error(), http.StatusInternalServerError)

		return
	}

	if len(res) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", ctJSON)
	enc := json.NewEncoder(w)

	if err := enc.Encode(res); err != nil {
		http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *MyShortener) findURL(key string) (string, error) {
	return myShortener.storage.LoadURL(key)
}

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

func (myShortener *MyShortener) authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		var usrID string

		errHandler := func(err error) {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		var token string
		cookie, err := r.Cookie("userID")
		if err != nil {
			errHandler(err)
		} else {
			token, err = openToken(cookie.String())
			if err != nil {
				log.Printf("failed to open passed uder ID %v", token)
			}
		}

		s := myShortener.storage
		val, err := s.LoadUser(token)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to restore session: \n%v", err.Error()), http.StatusInternalServerError)
		}

		if val != "" {
			usrID = val
		} else {
			usrID = uuid.New().String()
			newToken, err := sealToken(usrID)
			errHandler(err)
			err = s.StoreSession(usrID, newToken)
			errHandler(err)
		}

		setCookie(w, "userID", usrID)
		next.ServeHTTP(w, ctxWithSession(r, usrID))
	})
}

func openToken(token string) (string, error) {
	hexID, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}

	k, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	aesblock, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", err
	}

	nonce, err := newNonce(aesgcm)
	if err != nil {
		return "", err
	}

	dst, err := aesgcm.Open(nil, nonce, hexID, nil)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(dst), err
}

func sealToken(usrID string) (string, error) {
	k, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	aesblock, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", err
	}

	nonce, err := newNonce(aesgcm)
	if err != nil {
		return "", err
	}

	dst := aesgcm.Seal(nil, nonce, []byte(usrID), nil)

	return hex.EncodeToString(dst), err
}

func ctxWithSession(r *http.Request, usrID string) *http.Request {
	ctx := context.WithValue(r.Context(), ctxKeyUserID, usrID)
	return r.WithContext(ctx)
}

func setCookie(w http.ResponseWriter, name string, value string) {
	cookie := &http.Cookie{Name: name, Value: value}
	http.SetCookie(w, cookie)
}

func newNonce(aesgcm cipher.AEAD) ([]byte, error) {
	nonce := make([]byte, aesgcm.NonceSize())
	_, err := rand.Read(nonce)
	return nonce, err
}

//func (myShortener *MyShortener) Handlers() []router.HandlerDesc {
//	//return myShortener.Handlers
//
//	Handlers := []router.HandlerDesc{
//		{Method: "POST", Path: "/", Handler: http.HandlerFunc(s.makeShort), Middlewares: router.Middlewares(gzipMW, s.authMW)},
//		{Method: "POST", Path: "/api/shorten", Handler: http.HandlerFunc(s.makeShortJSON), Middlewares: router.Middlewares(gzipMW, s.authMW)},
//		{Method: "POST", Path: "/api/shorten/batch", Handler: http.HandlerFunc(s.shortenBatchJSON), Middlewares: router.Middlewares(gzipMW, s.authMW)},
//		{Method: "GET", Path: "/{id}", Handler: http.HandlerFunc(s.makeLong), Middlewares: router.Middlewares(gzipMW, s.authMW)},
//		{Method: "GET", Path: "/api/user/urls", Handler: http.HandlerFunc(s.makeLongByUser), Middlewares: router.Middlewares(gzipMW, s.authMW)},
//		{Method: "GET", Path: "/ping", Handler: http.HandlerFunc(s.pingStorage), Middlewares: router.Middlewares(s.authMW)},
//	}
//
//	for v := range Handlers{
//
//	}
//}

func (s *MyShortener) NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Route("/", s.defaultRoute())

	return r
}

func (s *MyShortener) defaultRoute() func(r chi.Router) {
	return func(r chi.Router) {

		r.Method("POST", "/", http.HandlerFunc(s.makeShort))
		r.Method("GET", "/{id}", http.HandlerFunc(s.makeLong))

		for _, route := range s.Handlers {
			r.With(route.Middlewares...).Method(route.Method, route.Path, route.Handler)
		}

		//Handlers := []router.HandlerDesc{
		//	{Method: "POST", Path: "/", Handler: http.HandlerFunc(s.makeShort), Middlewares: chi.Middlewares{gzipMW, s.authMW}},
		//	{Method: "POST", Path: "/api/shorten", Handler: http.HandlerFunc(s.makeShortJSON), Middlewares: chi.Middlewares{gzipMW, s.authMW}},
		//	{Method: "POST", Path: "/api/shorten/batch", Handler: http.HandlerFunc(s.shortenBatchJSON), Middlewares: chi.Middlewares{gzipMW, s.authMW}},
		//	{Method: "GET", Path: "/{id}", Handler: http.HandlerFunc(s.makeLong), Middlewares: chi.Middlewares{gzipMW, s.authMW}},
		//	{Method: "GET", Path: "/api/user/urls", Handler: http.HandlerFunc(s.makeLongByUser), Middlewares: chi.Middlewares{gzipMW, s.authMW}},
		//	{Method: "GET", Path: "/ping", Handler: http.HandlerFunc(s.pingStorage), Middlewares: chi.Middlewares{gzipMW, s.authMW}},
		//}
		//
		//for _, route := range Handlers {
		//	r.With(route.Middlewares...).Method(route.Method, route.Path, route.Handler)
		//}
	}
}

func (myShortener *MyShortener) FlushStorage() {
	if err := myShortener.storage.Flush(); err != nil {
		log.Fatal(err)
	}
}
