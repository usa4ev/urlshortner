package app

import (
	"encoding/json"
	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/storage"
	"io"
	"net/http"
	"strconv"
)

type (
	myShortener struct {
		urlMap  storage.StorageMap
		i       int
		baseURL string
	}
)

func NewShortener() *myShortener {

	return &myShortener{storage.NewStorage(), 0, configrw.ReadBaseURL()}
}

func (myShortener *myShortener) MakeShort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	URL, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	shortURL := shortURL(string(URL), myShortener)

	w.WriteHeader(http.StatusCreated)
	_, err = io.WriteString(w, shortURL)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *myShortener) MakeShortJSON(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		http.Error(w, "unsupported content type", http.StatusBadRequest)

		return
	}

	defer r.Body.Close()

	type urlreq struct {
		URL string `json:"url"`
	}

	message := urlreq{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&message)

	if err != nil {
		http.Error(w, "failed to decode message: "+err.Error(), http.StatusInternalServerError)

		return
	}

	type urlres struct {
		Result string `json:"result"`
	}

	res := urlres{}
	res.Result = shortURL(message.URL, myShortener)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	err = enc.Encode(res)

	if err != nil {
		http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *myShortener) MakeLong(w http.ResponseWriter, r *http.Request) {
	strid := r.URL.Path[1:]

	if strid == "" {
		http.Error(w, "id is required", http.StatusBadRequest)

		return
	}

	id, err := strconv.Atoi(strid)
	if err != nil {
		http.Error(w, "id must be an integer", http.StatusBadRequest)

		return
	}

	redirect := getPath(id, myShortener)
	if redirect == "" {
		http.NotFound(w, r)

		return
	}

	http.Redirect(w, r, redirect, http.StatusTemporaryRedirect)
}

func shortURL(url string, myShortener *myShortener) string {
	if myShortener.urlMap == nil {
		myShortener.urlMap = make(map[int]string)
	}

	for i, v := range myShortener.urlMap {
		if v == url {
			return myShortener.makeURL(i)
		}
	}

	myShortener.i++

	myShortener.urlMap[myShortener.i] = url

	err := storage.AppendStorage(myShortener.i, url)

	if err != nil {
		panic("failed to write new values to storage:" + err.Error())
	}

	return myShortener.makeURL(myShortener.i)
}

func (s *myShortener) makeURL(id int) string {

	return s.baseURL + strconv.Itoa(id)
}

func getPath(id int, myShortener *myShortener) string {
	return myShortener.urlMap[id]
}
