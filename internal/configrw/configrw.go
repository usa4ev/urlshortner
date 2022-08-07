package configrw

import (
	"flag"
	"os"
)

type Config struct {
	baseURL     string
	srvAddr     string
	storagePath string
}

func NewConfig() Config {
	s := Config{"http://localhost:8080", "localhost:8080", os.Getenv("HOME") + "/storage.csv"}

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

	if !flag.Parsed() {
		flag.StringVar(&s.baseURL, "b", s.baseURL, "base for short URLs")
		flag.StringVar(&s.srvAddr, "a", s.srvAddr, "the shortener service address")
		flag.StringVar(&s.storagePath, "f", s.storagePath, "path to a storage file")
		flag.Parse()
	}

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
