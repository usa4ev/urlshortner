package inmemory

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"sync"

	"github.com/usa4ev/urlshortner/internal/storage/storageerrors"

	"github.com/usa4ev/urlshortner/internal/storage/inmemory/filestorage"
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
		if val.(storer).deleted {
			return "", storageerrors.ErrURLGone
		}
		return val.(storer).url, nil
	}

	return "", fmt.Errorf("cannot find url by id %v", id)
}

func (s ims) LoadUrlsByUser(add func(id, url string), userID string) error {
	ch := make(chan item)

	s.findURLsByUser(ch, context.Background(), userID)

	for v := range ch {
		add(v.id, v.data.url)
	}

	return nil
}

func (s ims) StoreURL(id, url, userID string) error {
	var err error

	if _, ok := s.data.LoadOrStore(id, storer{url, userID, false}); ok {
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
		file, err := s.fileManager.OpnFileW()

		if err != nil {
			return err
		}
		writer := filestorage.NewWriter(file)

		f := func(key, value any) bool {
			err = writer.Write([]string{key.(string), value.(storer).url, value.(storer).userID, fmt.Sprintf("%t", value.(storer).deleted)})
			if err != nil {
				return false
			}
			return true
		}

		s.data.Range(f)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s ims) DeleteURLs(userID string, ids []string) error {

	ch := make(chan item)

	g, ctx := errgroup.WithContext(context.Background())

	s.findURLsByUser(ch, ctx, userID)

	g.Go(func() error {
		var err error
		for val := range ch {
			for i, v := range ids {
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

func (s ims) findURLsByUser(ch chan item, ctx context.Context, userID string) {
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
