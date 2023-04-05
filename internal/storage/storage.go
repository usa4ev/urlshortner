// Package storage provides an abstract interface
// for concrete storage implementations.
package storage

import (
	"context"
	"fmt"

	"github.com/usa4ev/urlshortner/internal/storage/database"
	"github.com/usa4ev/urlshortner/internal/storage/inmemory"
)

type (
	Storage struct {
		storerLoader
	}

	Pairs []Pair

	Pair struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}

	config interface {
		DBDSN() string
		StoragePath() string
	}

	storerLoader interface {
		LoadURL(id string) (string, error)
		LoadUrlsByUser(makeFunc func(id, url string), userID string) error
		StoreURL(id, url, userid string) error
		LoadUser(session string) (string, error)
		StoreSession(id, session string) error
		CountUsers() (int, error)
		CountURLs() (int, error)
		Flush() error
		DeleteURLs(userID string, ids []string) error
	}
)

// New returns new storage created using config
// to define the implementation.
func New(c config) (*Storage, error) {

	dsn := c.DBDSN()
	if dsn == "" {
		s, err := inmemory.New(c)
		if err != nil {

			return nil, fmt.Errorf("cannot create inmemory storage: %w", err)
		}

		return &Storage{s}, nil
	}

	db, err := database.New(dsn, context.Background())
	if err != nil {
		return nil, fmt.Errorf("cannot create database storage: %w", err)
	}

	return &Storage{db}, nil
}

// LoadByUser wraps LoadUrlsByUser storage method
//	to pass down the common appending function.
func (s Storage) LoadByUser(makeURL func(id string) string, userID string) (Pairs, error) {
	p := Pairs{}
	f := func(id, url string) {
		p = append(p, Pair{makeURL(id), url})
	}
	err := s.LoadUrlsByUser(f, userID)

	return p, err
}
