package filestorage

import (
	"bufio"
	"encoding/csv"
	"os"
	"sync"
)

type (
	fileStorage struct {
		filePath string
		writer   *bufio.Writer
		mx       *sync.Mutex
	}
	storer struct {
		url    string
		userID string
	}
)

func New(p string) *fileStorage {
	if p == "" {
		return nil
	}

	file, err := openFileW(p)
	if err != nil {
		panic("failed to open storage file:" + err.Error())
	}

	return &fileStorage{
		filePath: p,
		writer:   bufio.NewWriter(file),
		mx:       &sync.Mutex{},
	}
}

func (f fileStorage) ReadFile() (*sync.Map, error) {
	var s sync.Map

	file, err := os.OpenFile(f.filePath, os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0o777)
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

func (f fileStorage) Store(ss []string) error {
	writer := csv.NewWriter(f.writer)

	f.mx.Lock()
	err := writer.Write(ss)
	f.mx.Unlock()

	if err != nil {
		return err
	}

	return nil
}

func (f *fileStorage) Flush() error {
	return f.writer.Flush()
}

func openFileW(storagePath string) (*os.File, error) {
	file, err := os.OpenFile(storagePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o777)
	if err != nil {
		return nil, err
	}

	return file, nil
}
