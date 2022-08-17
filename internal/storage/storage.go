package storage

import (
	"context"

	"github.com/usa4ev/urlshortner/internal/storage/database"
	"github.com/usa4ev/urlshortner/internal/storage/inmemory"
)

type (
	Storage struct {
		storerLoader
	}

	pairs []Pair

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
		LoadUrlsByUser(func(id, url string), string) error
		StoreURL(id, url, userid string) error
		LoadUser(session string) (string, error)
		StoreSession(id, session string) error
		Flush() error
	}
)

func New(c config) *Storage {
	dsn := c.DBDSN()
	if dsn == "" {
		return &Storage{inmemory.New(c)}
	}

	return &Storage{database.New(dsn, context.Background())}
}

func (s Storage) LoadByUser(makeURL func(id string) string, userID string) (pairs, error) {
	p := pairs{}
	f := func(id, url string) {
		p = append(p, Pair{makeURL(id), url})
	}
	err := s.LoadUrlsByUser(f, userID)

	return p, err
}

func Ping(c config) error {
	return database.Pingdb(c.DBDSN())
}
