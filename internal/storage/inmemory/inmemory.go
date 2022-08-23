package inmemory

import (
	"fmt"
	"sync"

	"github.com/usa4ev/urlshortner/internal/storage/storageerrors"

	"github.com/usa4ev/urlshortner/internal/storage/inmemory/filestorage"
)

type (
	ims struct {
		data        *sync.Map
		sessions    *sync.Map
		fileManager fileStorer
	}

	storer struct {
		url    string
		userID string
	}
	config interface {
		StoragePath() string
	}
	fileStorer interface {
		ReadFile() (*sync.Map, error)
		Store([]string) error
		Flush() error
	}
)

func New(c config) ims {
	i := ims{sessions: &sync.Map{}}

	storagePath := c.StoragePath()
	if storagePath != "" {
		// setting up file storage if required
		i.fileManager = filestorage.New(storagePath)

		data, err := i.fileManager.ReadFile()
		if err != nil {
			panic(fmt.Errorf("failed to read from storage: %w", err))
		}

		i.data = data
	} else {
		i.data = &sync.Map{}
	}

	return i
}

func (s ims) LoadURL(id string) (string, error) {
	if val, ok := s.data.Load(id); ok {
		return val.(storer).url, nil
	}

	return "", fmt.Errorf("cannot find url by id %v", id)
}

func (s ims) LoadUrlsByUser(add func(id, url string), userid string) error {
	f := func(key, value any) bool {
		if value.(storer).userID == userid {
			add(key.(string), value.(storer).url)
		}

		return true
	}
	s.data.Range(f)

	return nil
}

func (s ims) StoreURL(id, url, userid string) error {
	var err error

	if _, ok := s.data.LoadOrStore(id, storer{url, userid}); !ok {
		if s.fileManager != nil {
			err = s.fileManager.Store([]string{url, id, userid})
		}
	} else {
		err = storageerrors.ErrConflict
	}

	return err
}

func (s ims) LoadUser(session string) (string, error) {
	val, ok := s.sessions.Load(session)
	if !ok {
		return "", nil
	}

	return val.(string), nil
}

func (s ims) StoreSession(id, session string) error {
	s.sessions.Store(session, id)

	return nil
}

func (s ims) Flush() error {
	if s.fileManager != nil {
		return s.fileManager.Flush()
	}

	return nil
}
