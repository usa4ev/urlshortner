package storage_test

import (
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/storage"
)

func TestStorage(t *testing.T) {
	config := configrw.NewConfig()
	storagePath := config.StoragePath()
	defer resetStorage(config.StoragePath())

	args := make(map[string]string)
	for i := 0; i <= 100; i++ {
		args["test"+strconv.Itoa(i)] = "test" + strconv.Itoa(i)
	}

	data := storage.NewStorage(storagePath)

	for k, v := range args {
		err := data.Append(k, v, storagePath)
		require.NoError(t, err, "failed to append storage")
	}

	require.NoError(t, data.Flush())

	dataRead := storage.NewStorage(storagePath)

	data.Range(func(k, v any) bool {
		vRead, ok := dataRead.Load(k.(string))
		assert.Equal(t, true, ok, "failed to read from storage")
		assert.Equal(t, vRead, v, "failed to read from storage")

		return true
	})
}

func resetStorage(path string) error {
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
