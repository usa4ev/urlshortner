package profiles

import (
	"fmt"
	"github.com/usa4ev/urlshortner/internal/config"
	"github.com/usa4ev/urlshortner/internal/storage/inmemory"
	"strconv"
	"testing"

	_ "net/http/pprof"
)

const (
	addr  = ":8080"
	cases = 1000
)

type item struct {
	id     string
	url    string
	userID string
}

func BenchmarkProf(b *testing.B) {
	data := make([]item, cases)

	for i := 0; i < cases; i++ {
		data[i] = item{
			strconv.Itoa(i),
			fmt.Sprintf("urls%v.com", i),
			fmt.Sprintf("user%v", i-i%3),
		}

	}

	vars := map[string]string{"FILE_STORAGE_PATH": ""}
	cfg := config.New(config.IgnoreOsArgs(), config.WithEnvVars(vars))
	storage, _ := inmemory.New(cfg)

	//store
	for _, v := range data {
		storage.StoreURL(v.id, v.url, v.userID)
	}

	// repeat to cover conflict cases
	for _, v := range data {
		storage.StoreURL(v.id, v.url, v.userID)
	}

	// load data
	for _, v := range data {
		storage.LoadURL(v.id)
	}

	// load data by user
	for _, v := range data {
		storage.LoadUrlsByUser(func(id, userID string) {}, v.url)
	}
}
