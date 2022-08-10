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
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/storage"
)

const (
	ctJSON       string     = "application/json"
	key          string     = "9cc1ee455a3363ffc504f40006f70d0c8276648a5d3eb3f9524e94d1b7a83aef"
	CtxKeyUserID contextKey = 0
)

type (
	MyShortener struct {
		storage  *storage.Storage
		Config   configrw.Config
		handlers []handler
		sessions *sync.Map
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
	contextKey int
)

func NewShortener() *MyShortener {
	s := &MyShortener{}
	s.Config = configrw.NewConfig()
	s.storage = storage.NewStorage(s.Config.StoragePath())
	s.sessions = &sync.Map{}
	s.handlers = []handler{
		{"POST", "/", http.HandlerFunc(s.makeShort), chi.Middlewares{gzipMW, s.authMW}},
		{"GET", "/{id}", http.HandlerFunc(s.makeLong), chi.Middlewares{gzipMW, s.authMW}},
		{"POST", "/api/shorten", http.HandlerFunc(s.makeShortJSON), chi.Middlewares{gzipMW, s.authMW}},
		{"GET", "/api/user/urls", http.HandlerFunc(s.makeLongByUser), chi.Middlewares{gzipMW, s.authMW}},
	}

	return s
}

func (myShortener *MyShortener) makeShort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	userID := r.Context().Value(CtxKeyUserID).(string)
	URL, err := readBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	shortURL := myShortener.shortenURL(string(URL), userID)

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

	userID := r.Context().Value(CtxKeyUserID).(string)
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

	res := urlres{myShortener.shortenURL(message.URL, userID)}

	w.Header().Set("Content-Type", ctJSON)
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)

	if err := enc.Encode(res); err != nil {
		http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *MyShortener) shortenURL(url, userID string) string {
	key := base64.RawURLEncoding.EncodeToString([]byte(url))
	storagePath := myShortener.Config.StoragePath()

	if err := myShortener.storage.Append(key, url, userID, storagePath); err != nil {
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

func (myShortener *MyShortener) makeLongByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(CtxKeyUserID)
	res := myShortener.storage.LoadByUser(userID.(string))
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

func findURL(key string, myShortener *MyShortener) (string, error) {
	if url, ok := myShortener.storage.LoadURL(key); ok {
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

func (myShortener *MyShortener) authMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		errHandler := func(err error) {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		var token string
		cookie, err := r.Cookie("userID")
		if err == nil {
			token, err = openToken(cookie.String())
			errHandler(err)
		}

		var usrID string

		s := myShortener.sessions
		if val, ok := s.Load(token); ok && val == token {
			usrID = val.(string)
		} else {
			usrID = uuid.New().String()
			newToken, err := sealToken(usrID)
			errHandler(err)
			myShortener.sessions.Store(newToken, usrID)
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

	nonce, err := newNonce(aesgcm, err)
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

	nonce, err := newNonce(aesgcm, err)
	if err != nil {
		return "", err
	}

	dst := aesgcm.Seal(nil, nonce, []byte(usrID), nil)

	return hex.EncodeToString(dst), err
}

func ctxWithSession(r *http.Request, usrID string) *http.Request {
	ctx := context.WithValue(r.Context(), CtxKeyUserID, usrID)
	return r.WithContext(ctx)
}

func setCookie(w http.ResponseWriter, name string, value string) {
	cookie := &http.Cookie{Name: name, Value: value}
	http.SetCookie(w, cookie)
}

func newNonce(aesgcm cipher.AEAD, err error) ([]byte, error) {
	nonce := make([]byte, aesgcm.NonceSize())
	_, err = rand.Read(nonce)
	return nonce, err
}

func (myShortener *MyShortener) Handlers() []handler {
	return myShortener.handlers
}

func (myShortener *MyShortener) FlushStorage() {
	if err := myShortener.storage.Flush(); err != nil {
		log.Fatal(err)
	}
}
