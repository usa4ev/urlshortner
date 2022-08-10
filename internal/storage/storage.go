package storage

import (
	"bufio"
	"encoding/csv"
	"os"
	"sync"
)

type (
	Storage struct {
		storageMap *sync.Map

		writer *bufio.Writer
		mx     *sync.Mutex
	}
	storer struct {
		url    string
		userID string
	}
	pairs []Pair
	Pair  struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
)

func NewStorage(storagePath string) *Storage {
	s, err := readStorage(storagePath)
	if err != nil {
		panic("failed to read from file storage: " + err.Error())
	}

	file, err := openStorageFile(storagePath)
	if err != nil {
		panic("failed to open storage file:" + err.Error())
	}

	storage := &Storage{s, bufio.NewWriter(file), &sync.Mutex{}}

	return storage
}

func openStorageFile(storagePath string) (*os.File, error) {
	file, err := os.OpenFile(storagePath, os.O_WRONLY|os.O_CREATE, 0o777)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func readStorage(storagePath string) (*sync.Map, error) {
	var s sync.Map

	// path is not set, quit wo error
	if storagePath == "" {
		return &s, nil
	}

	file, err := os.OpenFile(storagePath, os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0o777)
	if err != nil {
		return &s, err
	}

	defer file.Close()

	reader := csv.NewReader(file)

	strings, err := reader.ReadAll()
	if err != nil {
		return &s, err
	}

	for _, v := range strings {
		s.Store(v[0], storer{v[1], v[2]})
	}

	return &s, nil
}

func (s *Storage) Append(key, value, userID, storagePath string) error {
	if _, ok := s.storageMap.LoadOrStore(key, storer{value, userID}); ok {
		return nil
	}

	// path is not set, quit wo error
	if storagePath == "" {
		return nil
	}

	writer := csv.NewWriter(s.writer)

	s.mx.Lock()
	err := writer.Write([]string{key, value, userID})
	s.mx.Unlock()

	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) Flush() error {
	return s.writer.Flush()
}

func (s Storage) LoadURL(key string) (string, bool) {
	if val, ok := s.storageMap.Load(key); ok {
		return val.(storer).url, ok
	}

	return "", false
}

func (s Storage) LoadByUser(userID string) pairs {
	p := pairs{}
	f := func(key, value any) bool {
		if value.(storer).userID == userID {
			p = append(p, Pair{key.(string), value.(storer).url})
		}
		return true
	}

	s.Range(f)

	return p
}

func (s Storage) Range(f func(key, value any) bool) {
	s.storageMap.Range(f)
}
