package storage_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/usa4ev/urlshortner/internal/storage/database"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/usa4ev/urlshortner/internal/configrw"
	"github.com/usa4ev/urlshortner/internal/storage"
)

type storer struct {
	url    string
	userID string
}

func TestStorage(t *testing.T) {
	config := configrw.NewConfig()
	resetStorage(config.StoragePath(), config.DB_DSN())
	defer resetStorage(config.StoragePath(), config.DB_DSN())

	count := 100
	userID := "testUsr"

	args := make(map[string]storer)
	for i := 0; i < count; i++ {
		args["test"+strconv.Itoa(i)] = storer{"test" + strconv.Itoa(i), userID}
	}

	data := storage.New(config)

	for k, v := range args {
		err := data.StoreURL(k, v.url, v.userID)
		require.NoError(t, err, "failed to append storage")
	}

	require.NoError(t, data.Flush())

	/*dataRead := storage.New(config)

	data.Range(func(k, v any) bool {
		urlact, ok := dataRead.LoadURL(k.(string))
		urlexp, _ := data.LoadURL(k.(string))
		assert.Equal(t, true, ok, "failed to read from storage")
		assert.Equal(t, urlexp, urlact, "failed to read from storage")

		return true
	})*/

	/*	res := data.LoadByUser(userID)
		assert.Equal(t, len(res), count, "failed to read from storage by user")

		res = data.LoadByUser(userID + "0")
		assert.Equal(t, len(res), 0, "failed to read from storage by user")
	*/
}

func resetStorage(path, dsn string) error {
	// path is not set, quit wo error
	if path == "" {
		return nil
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	db := database.New(dsn, context.Background())
	defer db.Close()
	_, err := db.Query("TRUNCATE TABLE  urls")

	if err != nil {
		return err
	}

	_, err = db.Query("TRUNCATE TABLE users")
	if err != nil {
		return err
	}

	fmt.Println("storage reset successful")

	return nil
}
