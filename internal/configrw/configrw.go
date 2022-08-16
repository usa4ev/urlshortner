package configrw

import (
	"flag"
	"os"
)

type Config struct {
	baseURL     string
	srvAddr     string
	storagePath string
	db_DSN      string
}
type configOption func(o *configOptions)

func withOsArgs(osArgs []string) configOption {
	return func(o *configOptions) {
		o.osArgs = osArgs
	}
}

func withEnvVars(envVars map[string]string) configOption {
	return func(o *configOptions) {
		o.envVars = envVars
	}
}

func WithEnv() configOption {
	return func(o *configOptions) {
		o.allowEnv = true
	}
}

type configOptions struct {
	osArgs   []string
	envVars  map[string]string
	allowEnv bool
}

func NewConfig(opts ...configOption) Config {
	configOptions := &configOptions{
		osArgs: os.Args[1:],
		envVars: map[string]string{
			"BASE_URL":          os.Getenv("BASE_URL"),
			"SERVER_ADDRESS":    os.Getenv("SERVER_ADDRESS"),
			"FILE_STORAGE_PATH": os.Getenv("FILE_STORAGE_PATH"),
			"DATABASE_DSN":      os.Getenv("DATABASE_DSN"),
		},
	}

	for _, o := range opts {
		o(configOptions)
	}
	s := Config{"http://localhost:8080", "localhost:8080", os.Getenv("HOME") + "/storage.csv", "user=postgres password=postgres host=localhost port=5432 dbname=testdb"}

	if configOptions.allowEnv {
		if v := configOptions.envVars["BASE_URL"]; v != "" {
			s.baseURL = v
		}
		if v := configOptions.envVars["SERVER_ADDRESS"]; v != "" {
			s.srvAddr = v
		}
		if v := configOptions.envVars["FILE_STORAGE_PATH"]; v != "" {
			s.storagePath = v
		}
		if v := configOptions.envVars["DATABASE_DSN"]; v != "" {
			s.db_DSN = v
		}
	}

	fs := flag.NewFlagSet("myFS", flag.ContinueOnError)
	if !fs.Parsed() {
		fs.StringVar(&s.baseURL, "b", s.baseURL, "base for short URLs")
		fs.StringVar(&s.srvAddr, "a", s.srvAddr, "the shortener service address")
		fs.StringVar(&s.storagePath, "f", s.storagePath, "path to a storage file")
		fs.StringVar(&s.db_DSN, "d", s.db_DSN, "db connection path")
		//fs.Parse(os.Args[1:])
		fs.Parse(configOptions.osArgs)
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

func (c Config) DB_DSN() string {
	return c.db_DSN
}
