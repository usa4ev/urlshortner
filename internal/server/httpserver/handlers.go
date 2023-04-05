package httpserver

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/usa4ev/urlshortner/internal/server/httpserver/middleware"
	"github.com/usa4ev/urlshortner/internal/storage/database"
	"github.com/usa4ev/urlshortner/internal/storage/storageerrors"
)

// pingStorage returns error code as a response if failed to connect to database storage.
func (srv *Server) pingStorage(w http.ResponseWriter, r *http.Request) {
	err := database.Pingdb(srv.cfg.DBDSN())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

// makeShort responds with a short URL as a plain text.
func (srv *Server) makeShort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	userID := r.Context().Value(middleware.CtxKeyUserID).(string)
	originalURL, err := readBody(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	id, url := srv.shortener.ShortenURL(string(originalURL))
	err = srv.shortener.StoreURL(id, string(originalURL), userID)
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

// makeShortJSON responds with a short URL as a urlres JSON structure.
func (srv *Server) makeShortJSON(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != ctJSON {
		http.Error(w, "unsupported content type", http.StatusBadRequest)

		return
	}

	defer r.Body.Close()

	var userID string
	if rawUserID := r.Context().Value(middleware.CtxKeyUserID); rawUserID != nil {
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
	id, url := srv.shortener.ShortenURL(message.URL)
	res := urlres{url}
	err = srv.shortener.StoreURL(id, message.URL, userID)
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

// shortenBatchJSON responds with a short URL as an urlwidres JSON structure
// setting correlation_id with respective value from recieved urlwid.
func (srv *Server) shortenBatchJSON(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != ctJSON {
		http.Error(w, "unsupported content type", http.StatusBadRequest)

		return
	}

	defer r.Body.Close()

	var userID string
	if rawUserID := r.Context().Value(middleware.CtxKeyUserID); rawUserID != nil {
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
		id, url := srv.shortener.ShortenURL(v.OriginalURL)
		res = append(res, urlwidres{v.CorrelationID, url})
		err = srv.shortener.StoreURL(id, v.OriginalURL, userID)
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

// makeLong load URL from storage by ID and, if found, redirects client.
func (srv *Server) makeLong(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[1:]
	redirect, err := srv.shortener.FindURL(id)
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

// makeLongByUser responds with encoded JSON collection
// that contains all the URLs uploaded by the user.
func (srv *Server) makeLongByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.CtxKeyUserID)
	res, err := srv.shortener.LoadByUser(userID.(string))
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

// deleteBatch receives list of URL ids that need to be deleted.
// Deletion is executed asynchronously.
func (srv *Server) deleteBatch(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != ctJSON {
		http.Error(w, "unsupported content type", http.StatusBadRequest)

		return
	}

	defer r.Body.Close()

	var userID string
	if rawUserID := r.Context().Value(middleware.CtxKeyUserID); rawUserID != nil {
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

	err = srv.shortener.DeleteURLs(userID, message)

	if err != nil {
		http.Error(w, "deletion failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// stats returns JSON encoded statsData.
// Request will be accepted from trusted subnet only.
func (srv *Server) stats(w http.ResponseWriter, r *http.Request) {
	rawXFF := r.Header.Get("X-Real-IP")
	xff := strings.Split(rawXFF, ",")

	//take the rightmost IP
	ipstr := xff[len(xff)-1]
	ip := net.ParseIP(ipstr)

	//parse subnet string
	_, net, err := net.ParseCIDR(srv.cfg.TrustedSubnet())
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("failed to parse trusted subnet from config: %v",
				srv.cfg.TrustedSubnet()),
			http.StatusInternalServerError)
		return
	}

	//check if ip belongs subnet
	if ip == nil || !net.Contains(ip) {
		http.Error(w, "", http.StatusForbidden)
		return
	}

	//use SingleFlight
	urls, err, _ := srv.sfgr.Do("CountURLs",
		func() (interface{}, error) {
			return srv.shortener.CountURLs()
		})
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("failed to get data from storage: %v",
				err.Error()),
			http.StatusInternalServerError)
		return
	}

	users, err, _ := srv.sfgr.Do("CountUsers",
		func() (interface{}, error) {
			return srv.shortener.CountUsers()
		})

	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("failed to get data from storage: %v",
				err.Error()),
			http.StatusInternalServerError)
		return
	}

	data := statsData{
		Urls:  urls.(int),
		Users: users.(int),
	}

	buf := bytes.NewBuffer(nil)
	err = json.NewEncoder(buf).Encode(data)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("failed to encode stats data response: %v", data),
			http.StatusInternalServerError)
		return
	}

	w.Write(buf.Bytes())
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
