// Package shortener implements handling methods
// used for URL-shortner service.
package shortener

import (
	"encoding/base64"

	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/storage"
)

type Shortener interface {
	ShortenURL(url string) (string, string) // ShortenURL returns a short id and a short URL.
	StoreURL(id, url, userID string) error
	FindURL(key string) (string, error)
	LoadByUser(userID string) (storage.Pairs, error)
	DeleteURLs(userID string, ids []string) error
	CountUsers() (int, error)
	CountURLs() (int, error)
	FlushStorage() error
}
type (
	MyShortener struct {
		storage *storage.Storage
		config  *config.Config
	}
)

func NewShortener(c *config.Config, s *storage.Storage) *MyShortener {
	myShortener := &MyShortener{}
	myShortener.config = c
	myShortener.storage = s

	return myShortener
}

// ShortenURL returns a short id and a short URL.
func (myShortener *MyShortener) ShortenURL(url string) (string, string) {
	// ToDo: the way it works results in URLs become rather longer than shorter.
	id := base64.RawURLEncoding.EncodeToString([]byte(url))
	return id, myShortener.makeURL(id)
}

func (myShortener *MyShortener) StoreURL(id, url, userID string) error {
	return myShortener.storage.StoreURL(id, url, userID)
}

func (myShortener *MyShortener) makeURL(id string) string {
	return myShortener.config.BaseURL() + "/" + id
}

func (myShortener *MyShortener) FindURL(key string) (string, error) {
	return myShortener.storage.LoadURL(key)
}

func (myShortener *MyShortener) FlushStorage() error {
	return myShortener.storage.Flush()
}

func (myShortener *MyShortener) LoadByUser(userID string) (storage.Pairs, error) {
	return myShortener.storage.LoadByUser(myShortener.makeURL, userID)
}

func (myShortener *MyShortener) DeleteURLs(userID string, ids []string) error {
	return myShortener.storage.DeleteURLs(userID, ids)
}

func (myShortener *MyShortener) CountURLs() (int, error) {
	return myShortener.storage.CountURLs()
}

func (myShortener *MyShortener) CountUsers() (int, error) {
	return myShortener.storage.CountUsers()
}
