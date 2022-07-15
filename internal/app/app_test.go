package app

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShortURL(t *testing.T) {
	storage := NewStorage()
	args := []struct {
		url string
		id  int
	}{
		{
			"http://test/test1",
			1,
		},
		{
			"http://test/test2",
			2,
		},
		{
			"http://test/test3",
			3,
		},
	}
	for _, v := range args {
		ShortURL(v.url, storage)
	}

	t.Run("Storage fill test", func(t *testing.T) {
		assert.Equal(t, 3, storage.id, "storage counter expected to be %c, got %v", len(args), storage.id)

		for _, v := range args {
			assert.Equal(t, v.url, storage.urlMap[v.id], "storage.url expected to be %c, got %v", v.url, storage.urlMap[v.id])
		}
	})
}

func TestGetPath(t *testing.T) {
	t.Run("Storage reader test", func(t *testing.T) {
		s := "test"
		storage := NewStorage()
		storage.urlMap[1] = s
		assert.Equal(t, s, GetPath(1, storage), "storage output expected to be %c, got %v", s, GetPath(1, storage))
	})
}
