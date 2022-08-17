package database

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/usa4ev/urlshortner/internal/configrw"
	"testing"
)

func resetStorage(dsn string) error {
	// path is not set, quit wo error
	db := New(dsn, context.Background())
	defer db.Close()

	rows, err := db.Query("DROP TABLE urls, users")

	if rows.Err() != nil {
		return rows.Err()
	}

	return err
}

func TestPingdb(t *testing.T) {
	config := configrw.NewConfig(configrw.IgnoreOsArgs())
	defer resetStorage(config.DBDSN())
	//db := New(config.DBDSN(), context.Background()))

	type args struct {
		dsn string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"with valid dsn",
			args{config.DBDSN()},
			false,
		},
		{
			"with invalid dsn",
			args{"wrong.dsn"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Pingdb(tt.args.dsn); (err != nil) != tt.wantErr {
				t.Errorf("Pingdb() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_ims_StoreLoad(t *testing.T) {
	config := configrw.NewConfig(configrw.IgnoreOsArgs())
	resetStorage(config.DBDSN())
	testUserID := "testuser"
	testSession := "testSession"

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

	storage := New(config.DBDSN(), context.Background())

	t.Run("Store user info StoreSession()", func(t *testing.T) {
		if err := storage.StoreSession(testUserID, testSession); err != nil {
			require.NoError(t, err, "Error occurred when tried to store user info")
		}
	})

	t.Run("Load user ID LoadUser()", func(t *testing.T) {

		got, err := storage.LoadUser(testSession)
		if err != nil {
			require.NoError(t, err, "LoadUser() error")
		}
		assert.Equal(t, testUserID, got, "got wrong user id by id %v", testSession)
	})

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
		assert.Equal(t, c, len(p), "got wrong number of url by user %v", testUserID)
	})

}
