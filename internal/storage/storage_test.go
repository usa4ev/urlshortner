package storage_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/storage"
)

func TestStorage(t *testing.T) {
	err := resetStorage()
	require.NoError(t, err, "failed to reset storage")
	defer resetStorage()

	args := storage.Storage{"test": "test", "test1": "test1", "test2": "test2"}

	for k, v := range args {
		err := storage.AppendStorage(k, v)
		require.NoError(t, err, "failed to append storage")
	}

	storage := storage.NewStorage()

	for k, v := range args {
		assert.Equal(t, storage[k], v, "failed to read from storage")
	}
}

func resetStorage() error {
	path := configrw.ReadStoragePath()

	// path is not set, quit wo error
	if path == "" {
		return nil
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// path does not exist, nothing to delete
		return nil
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	return nil
}
