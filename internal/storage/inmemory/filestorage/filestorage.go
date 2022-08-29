package filestorage

import (
	"encoding/csv"
	"os"
	"strconv"
	"sync"
)

type (
	FileStorage struct {
		filePath string
	}
	storer struct {
		url     string
		userID  string
		deleted bool
	}
	Writer interface {
		Write([]string) error
	}
)

func New(p string) *FileStorage {
	if p == "" {
		return nil
	}

	return &FileStorage{
		filePath: p,
	}
}

func (f FileStorage) ReadFile() (*sync.Map, error) {
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
		deleted, err := strconv.ParseBool(v[3])
		if err != nil {
			return &s, err
		}

		s.Store(v[0], storer{v[1], v[2], deleted})
	}

	return &s, nil
}

func NewWriter(file *os.File) Writer {
	return csv.NewWriter(file)
}

func (f FileStorage) OpnFileW() (*os.File, error) {
	return openFileW(f.filePath)
}

func openFileW(storagePath string) (*os.File, error) {
	file, err := os.OpenFile(storagePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o777)
	if err != nil {
		return nil, err
	}

	return file, nil
}
