// Package inmemory implements a thread-safe
// in-memory storage via sync.map to store sessions and URLs.
// It can, optionally, use file storage to load previously
// saved values if the path to a storage file is defined in config.
package inmemory

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/usa4ev/urlshortner/internal/storage/inmemory/filestorage"
	"github.com/usa4ev/urlshortner/internal/storage/storageerrors"
)

type (
	ims struct {
		data        *sync.Map
		sessions    *sync.Map
		fileManager *filestorage.FileStorage
	}

	item struct {
		id   string
		data storer
	}

	storer struct {
		url     string
		userID  string
		deleted bool
	}

	config interface {
		StoragePath() string
	}
)

// New creates new storage and load data from file if necessary.
func New(c config) (ims, error) {
	var i ims
	storagePath := c.StoragePath()
	if storagePath != "" {
		// setting up file storage if required
		i.fileManager = filestorage.New(storagePath)

		data, err := i.fileManager.ReadFile()
		if err != nil {

			return i, fmt.Errorf("failed to read from storage: %w", err)
		}

		i.data = data
	} else {
		i.data = &sync.Map{}
	}

	i.sessions = &sync.Map{}

	return i, nil
}

// LoadURL returns long URL loading one by id.
func (s ims) LoadURL(id string) (string, error) {
	if val, ok := s.data.Load(id); ok {
		if val.(storer).deleted {
			return "", storageerrors.ErrURLGone
		}

		return val.(storer).url, nil
	}

	return "", fmt.Errorf("cannot find url by id %v", id)
}

// LoadUrlsByUser uses received function to return loaded values.
func (s ims) LoadUrlsByUser(add func(id, url string), userID string) error {
	ch := make(chan item)

	s.findURLsByUser(context.Background(), ch, userID)

	for v := range ch {
		add(v.id, v.data.url)
	}

	return nil
}

// StoreURL adds url to the data and returns an error if the key is not unique.
func (s ims) StoreURL(id, url, userID string) error {
	if _, ok := s.data.LoadOrStore(id, storer{url, userID, false}); ok {
		return storageerrors.ErrConflict
	}

	return nil
}

// LoadUser user ID from the sessions map using passed token as a key.
func (s ims) LoadUser(session string) (string, error) {
	val, ok := s.sessions.Load(session)
	if !ok {
		return "", nil
	}

	return val.(string), nil
}

// StoreSession adds user ID to the sessions map.
func (s ims) StoreSession(id, session string) error {
	s.sessions.Store(session, id)

	return nil
}

// Flush writes data from the storage to a file if file manager is set.
func (s ims) Flush() error {
	if s.fileManager != nil {
		file, err := s.fileManager.OpnFileW()
		if err != nil {
			return err
		}

		writer := filestorage.NewWriter(file)

		f := func(key, value any) bool {
			err = writer.Write([]string{key.(string), value.(storer).url, value.(storer).userID, fmt.Sprintf("%t", value.(storer).deleted)})

			return err == nil
		}

		s.data.Range(f)

		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteURLs deletes URLs if they were uploaded by the user with userID.
func (s ims) DeleteURLs(userID string, ids []string) error {
	ch := make(chan item)

	g, ctx := errgroup.WithContext(context.Background())

	s.findURLsByUser(ctx, ch, userID)

	g.Go(func() error {
		var err error
		// iterate items found by userID
		for val := range ch {
			for i, v := range ids {
				// if items id matches one from the ids slice
				// we can safely delete it and remove the id from the slice
				if val.id == v {
					s.data.Store(val.id, storer{val.data.url, val.data.userID, true})
					ids = append(ids[:i], ids[i+1:]...)

					break
				}
			}
		}

		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (s ims) findURLsByUser(ctx context.Context, ch chan item, userID string) {
	go func() {
		defer func(ch chan item) {
			close(ch)
		}(ch)

		f := func(key, value any) bool {
			if value.(storer).userID == userID && !value.(storer).deleted {
				select {
				case <-ctx.Done():
					return false
				default:
					ch <- item{key.(string), value.(storer)}
				}
			}

			return true
		}

		s.data.Range(f)
	}()
}
