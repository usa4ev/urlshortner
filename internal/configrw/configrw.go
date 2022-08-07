package configrw

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
	baseURL     string
	srvAddr     string
	storagePath string
}

func NewConfig() Config {
	s := Config{"http://localhost:8080", "127.0.0.1:8080", os.Getenv("HOME") + "/storage.csv"}

	// setting up default values first
	envVars := map[string]*string{
		"BASE_URL":          &s.baseURL,
		"SERVER_ADDRESS":    &s.srvAddr,
		"FILE_STORAGE_PATH": &s.storagePath,
	}
	for key, envVar := range envVars {
		if v, ok := os.LookupEnv(key); ok {
			*envVar = v
		}
	}

	fs := flag.NewFlagSet("myFS", flag.PanicOnError)
	if !fs.Parsed() {
		fs.StringVar(&s.baseURL, "b", s.baseURL, "base for short URLs")
		fs.StringVar(&s.srvAddr, "a", s.srvAddr, "the shortener service address")
		fs.StringVar(&s.storagePath, "f", s.storagePath, "path to a storage file")
		fs.Parse([]string{"b", "a", "f"})
	}

	fmt.Println(s.storagePath)

	return s
}

func (c Config) BaseURL() string {
	return c.baseURL
}

func (c Config) SrvAddr() string {
	return c.srvAddr
}

func (c Config) StoragePath() string {
	return c.storagePath
}
