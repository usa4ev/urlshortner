package storage

import (
	"encoding/csv"
	"github.com/usa4ev/urlshortner/internal/configrw"
	"os"
	"strconv"
)

type StorageMap map[int]string

func NewStorage() StorageMap {
	s := make(StorageMap)
	err := readStorage(s)
	if err != nil {
		panic("failed to read from file storage: " + err.Error())
	}
	return s
}

func readStorage(s StorageMap) error {
	path := configrw.ReadStoragePath()

	//path is not set, quit wo error
	if path == "" {
		return nil
	}

	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0777)
	defer file.Close()

	if err != nil {
		return err
	}

	reader := csv.NewReader(file)
	strings, err := reader.ReadAll()

	for _, v := range strings {
		i, err := strconv.Atoi(v[0])
		if err != nil {
			return err
		}

		s[i] = v[1]
	}

	return nil
}

func AppendStorage(key int, value string) error {
	path := configrw.ReadStoragePath()

	//path is not set, quit wo error
	if path == "" {
		return nil
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	defer file.Close()

	if err != nil {
		return err
	}

	writer := csv.NewWriter(file)
	err = writer.Write([]string{strconv.Itoa(key), value})

	if err != nil {
		return err
	}

	writer.Flush()

	return writer.Error()
}
