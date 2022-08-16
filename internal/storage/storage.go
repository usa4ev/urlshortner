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
		DB_DSN() string
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

	fileStorage interface {
		LoadAll()
		Append()
		Flush() error
	}
)

func New(c config) *Storage {
	dsn := c.DB_DSN()
	if dsn == "" {
		return &Storage{inmemory.New(c)}
	} else {
		return &Storage{database.New(dsn, context.Background())}
	}
}

/*func (s *Storage) Store(id, url, userid string) error {
	return s.StoreURL(id, url, userid)
}*/
/*
func (s *Storage) Flush() error {
	return s.Flush()
}*/
/*
func (s Storage) LoadURL(id string) (string, error) {
	return s.LoadURL(id)
}*/

func (s Storage) LoadByUser(makeURL func(id string) string, userID string) (pairs, error) {
	p := pairs{}
	f := func(id, url string) {
		p = append(p, Pair{makeURL(id), url})
	}
	err := s.LoadUrlsByUser(f, userID)

	return p, err
}

func (p *pairs) add(shortURL, originalURL string) {
	*p = append(*p, Pair{shortURL, originalURL})
}

func Ping(c config) error {
	return database.Pingdb(c.DB_DSN())
}
