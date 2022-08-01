package storage

import (
	"encoding/csv"
	"os"

	"github.com/usa4ev/urlshortner/internal/configrw"
)

type Storage map[string]string

func NewStorage() Storage {
	s := make(Storage)

	if err := readStorage(s); err != nil {
		panic("failed to read from file storage: " + err.Error())
	}

	return s
}

func readStorage(s Storage) error {
	path := configrw.ReadStoragePath()

	// path is not set, quit wo error
	if path == "" {
		return nil
	}

	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o777)
	if err != nil {
		return err
	}

	defer file.Close()

	reader := csv.NewReader(file)

	strings, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for _, v := range strings {
		key := v[0]
		s[key] = v[1]
	}

	return nil
}

func AppendStorage(key, value string) error {
	path := configrw.ReadStoragePath()

	// path is not set, quit wo error
	if path == "" {
		return nil
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o777)
	if err != nil {
		return err
	}

	defer file.Close()

	writer := csv.NewWriter(file)
	err = writer.Write([]string{key, value})

	if err != nil {
		return err
	}

	writer.Flush()

	return writer.Error()
}
