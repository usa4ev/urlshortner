package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

type MyShortener struct {
	urlMap map[int]string
	i      int
}

func (myShortener *MyShortener) MakeShort(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	URL, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	shortURL := shortURL(string(URL), r.Host, myShortener)

	w.WriteHeader(http.StatusCreated)
	_, err = io.WriteString(w, shortURL)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *MyShortener) MakeShortJSON(w http.ResponseWriter, r *http.Request) {
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
	res.Result = shortURL(message.URL, r.Host, myShortener)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	err = enc.Encode(res)

	if err != nil {
		http.Error(w, "failed to encode message: "+err.Error(), http.StatusInternalServerError)

		return
	}
}

func (myShortener *MyShortener) MakeLong(w http.ResponseWriter, r *http.Request) {
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

func shortURL(url string, host string, myShortener *MyShortener) string {
	if myShortener.urlMap == nil {
		myShortener.urlMap = make(map[int]string)
	}

	for i, v := range myShortener.urlMap {
		if v == url {
			return makeURL(host, i)
		}
	}

	myShortener.i++

	myShortener.urlMap[myShortener.i] = url

	return makeURL(host, myShortener.i)
}

func makeURL(host string, id int) string {
	return "http://" + host + "/" + strconv.Itoa(id)
}

func getPath(id int, myShortener *MyShortener) string {
	return myShortener.urlMap[id]
}
