package inmemory_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/storage/inmemory"
)

func resetStorage(path string) error {
	// path is not set, quit wo error
	if path == "" {
		return nil
	}

	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		// ignore
	} else if !errors.Is(err, os.ErrNotExist) && err != nil {
		return err
	} else {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	return nil
}

func Test_ims_StoreLoadURL(t *testing.T) {
	config := config.New(config.IgnoreOsArgs())
	defer resetStorage(config.StoragePath())

	testUserID := "testuser"

	type args struct {
		id,
		url,
		userid string
	}

	tests := []args{
		{
			"1",
			"ya.ru",
			testUserID,
		},
		{
			"2",
			"go.com",
			testUserID,
		},
	}

	storage, err := inmemory.New(config)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run("Strore URL's", func(t *testing.T) {
			if err := storage.StoreURL(tt.id, tt.url, tt.userid); err != nil {
				require.NoError(t, err, "Error occurred when tried to store URL")
			}
		})
	}

	for _, tt := range tests {
		t.Run("Load URL's", func(t *testing.T) {
			got, err := storage.LoadURL(tt.id)
			if err != nil {
				require.NoError(t, err, "LoadURL() error")
			}
			assert.Equal(t, tt.url, got, "got wrong URL by id %v", tt.id)
		})
	}

	t.Run("Load URL's by user", func(t *testing.T) {
		type pair struct{ shortURL, originalURL string }
		p := []pair{}
		f := func(shortURL, originalURL string) {
			p = append(p, pair{shortURL, originalURL})
		}

		c := 0
		for _, v := range tests {
			if v.userid == testUserID {
				c++
			}
		}

		err := storage.LoadUrlsByUser(f, testUserID)
		if err != nil {
			require.NoError(t, err, "LoadUrlsByUser() error")
		}
		assert.Equal(t, c, len(p), "got wrong number of url's by user %v", testUserID)
	})
}

func Example() {
	config := config.New(config.IgnoreOsArgs())

	// new storage
	storage, _ := inmemory.New(config)

	// url data to store
	id := "1" // internal identifier
	userID := "New User"
	url := "foo.com"

	// store data
	storage.StoreURL(id, url, userID)

	// load data
	got, _ := storage.LoadURL(id)

	fmt.Printf("Stored %v, got %v", url, got)

	// session data to store
	userID = "2"
	token := "jkSDFg8923ur"

	// store session
	storage.StoreSession(userID, token)

	// load userID
	got, _ = storage.LoadUser(token)

	fmt.Printf("Stored %v, got %v", userID, got)
}

func Test_ims_StoreLoadUserInfo(t *testing.T) {
	config := config.New(config.IgnoreOsArgs())
	defer resetStorage(config.StoragePath())

	type args struct {
		session,
		userid string
	}

	tests := []args{
		{
			"token",
			"testuser",
		},
	}

	storage, err := inmemory.New(config)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run("Store user info", func(t *testing.T) {
			if err := storage.StoreSession(tt.userid, tt.session); err != nil {
				require.NoError(t, err, "Error occurred when tried to store user info")
			}
		})
	}

	for _, tt := range tests {
		t.Run("Load user ID", func(t *testing.T) {
			got, err := storage.LoadUser(tt.session)
			if err != nil {
				require.NoError(t, err, "LoadUser() error")
			}
			assert.Equal(t, tt.userid, got, "got wrong user id by id %v", tt.session)
		})
	}
}

func Test_ims_DeleteURLs(t *testing.T) {
	config := config.New(config.IgnoreOsArgs())
	defer resetStorage(config.StoragePath())

	testUserID := "testuser"

	type args struct {
		id,
		url,
		userid string
	}

	tests := []args{
		{
			"1",
			"ya.ru",
			testUserID,
		},
		{
			"2",
			"go.com",
			testUserID,
		},
		{
			"3",
			"go.org",
			"different user",
		},
	}

	storage, err := inmemory.New(config)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run("Strore URL's", func(t *testing.T) {
			if err := storage.StoreURL(tt.id, tt.url, tt.userid); err != nil {
				require.NoError(t, err, "Error occurred when tried to store URL")
			}
		})
	}

	t.Run("Delete URL's", func(t *testing.T) {
		type pair struct{ shortURL, originalURL string }
		p := []pair{}
		f := func(shortURL, originalURL string) {
			p = append(p, pair{shortURL, originalURL})
		}

		ids := make([]string, len(tests))
		for i, tt := range tests {
			ids[i] = tt.id
		}

		err := storage.DeleteURLs(testUserID, ids)
		require.NoError(t, err)

		err = storage.LoadUrlsByUser(f, testUserID)
		if err != nil {
			require.NoError(t, err, "LoadUrlsByUser() error")
		}
		assert.Equal(t, 0, len(p), "got wrong number of url's by user %v", testUserID)
	})
}
