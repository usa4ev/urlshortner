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
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/router"
	"github.com/usa4ev/urlshortner/internal/storage"
	"github.com/usa4ev/urlshortner/internal/storage/storageerrors"

	"github.com/google/uuid"
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
		config   *config.Config
		handlers []router.HandlerDesc
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

func NewShortener(c *config.Config, s *storage.Storage) *MyShortener {
	myShortener := &MyShortener{}
	myShortener.config = c
	myShortener.storage = s
	myShortener.handlers = []router.HandlerDesc{
		{Method: "POST", Path: "/", Handler: http.HandlerFunc(myShortener.makeShort), Middlewares: chi.Middlewares{gzipMW, myShortener.authMW}},
		{Method: "GET", Path: "/{id}", Handler: http.HandlerFunc(myShortener.makeLong), Middlewares: chi.Middlewares{gzipMW, myShortener.authMW}},
		{Method: "POST", Path: "/api/shorten", Handler: http.HandlerFunc(myShortener.makeShortJSON), Middlewares: chi.Middlewares{gzipMW, myShortener.authMW}},
		{Method: "POST", Path: "/api/shorten/batch", Handler: http.HandlerFunc(myShortener.shortenBatchJSON), Middlewares: chi.Middlewares{gzipMW, myShortener.authMW}},
		{Method: "GET", Path: "/api/user/urls", Handler: http.HandlerFunc(myShortener.makeLongByUser), Middlewares: chi.Middlewares{gzipMW, myShortener.authMW}},
		{Method: "DELETE", Path: "/api/user/urls", Handler: http.HandlerFunc(myShortener.deleteBatch), Middlewares: chi.Middlewares{gzipMW, myShortener.authMW}},
		{Method: "GET", Path: "/ping", Handler: http.HandlerFunc(myShortener.pingStorage), Middlewares: chi.Middlewares{gzipMW, myShortener.authMW}},
	}

	return myShortener
}

func (myShortener *MyShortener) pingStorage(w http.ResponseWriter, r *http.Request) {
	err := database.Pingdb(myShortener.config.DBDSN())
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
		if errors.Is(err, storageerrors.ErrConflict) {
			w.WriteHeader(http.StatusConflict)

			_, err = io.WriteString(w, url)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)

				return
			}

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
		if errors.Is(err, storageerrors.ErrConflict) {
			w.Header().Set("Content-Type", ctJSON)
			w.WriteHeader(http.StatusConflict)
			if err := enc.Encode(res); err != nil {
				http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

				return
			}

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
		err = myShortener.storeURL(id, v.OriginalURL, userID)
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
	return myShortener.config.BaseURL() + "/" + id
}

func (myShortener *MyShortener) makeLong(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[1:]
	redirect, err := myShortener.findURL(id)
	switch {
	case errors.Is(err, storageerrors.ErrURLGone):
		http.Error(w, err.Error(), http.StatusGone)

		return
	case err != nil && !errors.Is(err, storageerrors.ErrURLGone):
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	case redirect == "":

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

func (myShortener *MyShortener) deleteBatch(w http.ResponseWriter, r *http.Request) {
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
		return
	}

	message := make([]string, 0)
	dec := json.NewDecoder(bytes.NewBuffer(body))

	if err := dec.Decode(&message); err != nil {
		http.Error(w, "failed to decode message: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = myShortener.storage.DeleteURLs(userID, message)

	if err != nil {
		http.Error(w, "deletion failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (myShortener *MyShortener) findURL(key string) (string, error) {
	return myShortener.storage.LoadURL(key)
}

func (myShortener *MyShortener) FlushStorage() error {
	return myShortener.storage.Flush()
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
				return
			}
		}
		var token, openToken string

		cookie, err := r.Cookie("userID")
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			errHandler(err)
		} else if err == nil {
			token = cookie.Value
			openToken, err = unsealToken(token)
			if err != nil {
				log.Printf("failed to open passed token %v", openToken)
			}
		}

		s := myShortener.storage
		val, err := s.LoadUser(openToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to restore session: \n%v", err.Error()), http.StatusInternalServerError)
		}

		if val != "" {
			usrID = val
		} else {
			usrID = uuid.New().String()
			openToken, err = generateRandom(16)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to create token for user ID: %v \n%v\n", usrID, err.Error()), http.StatusInternalServerError)
			}

			token, err = sealToken(openToken)
			errHandler(err)
			err = s.StoreSession(usrID, openToken)
			errHandler(err)
		}

		setCookie(w, "userID", token)
		next.ServeHTTP(w, ctxWithSession(r, usrID))
	})
}

func generateRandom(size int) (string, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func unsealToken(token string) (string, error) {
	hexID, err := hex.DecodeString(token)
	if err != nil {
		return "", err
	}

	k, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	aesblock, err := aes.NewCipher(k[:32])
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", err
	}

	nonce := k[len(k)-aesgcm.NonceSize():]

	dst, err := aesgcm.Open(nil, nonce, hexID, nil)
	if err != nil {
		return "", err
	}

	return string(dst), err
}

func sealToken(usrID string) (string, error) {
	k, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	aesblock, err := aes.NewCipher(k[:32])
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", err
	}

	nonce := k[len(k)-aesgcm.NonceSize():]

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

func (myShortener *MyShortener) Handlers() []router.HandlerDesc {
	return myShortener.handlers
}
