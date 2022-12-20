package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
)

type Config struct {
	baseURL     string
	srvAddr     string
	storagePath string
	dbDSN       string
	sslPath     string
	useTLS      bool
}
type (
	configOption func(o *configOptions)

	configOptions struct {
		osArgs        []string
		envVars       map[string]string
		ignoreOsArgs  bool
		ignoreCfgFile bool
		filePath      string
	}
)

func withOsArgs(osArgs []string) configOption {
	return func(o *configOptions) {
		o.osArgs = osArgs
	}
}

func WithEnvVars(envVars map[string]string) configOption {
	return func(o *configOptions) {
		o.envVars = envVars
	}
}

func IgnoreOsArgs() configOption {
	return func(o *configOptions) {
		o.ignoreOsArgs = true
	}
}

func WithFile(path string) configOption {
	return func(o *configOptions) {
		o.filePath = path
	}
}

func New(opts ...configOption) *Config {
	var tlsModeSet bool
	configOptions := &configOptions{
		osArgs: os.Args[1:],
		envVars: map[string]string{
			"BASE_URL":          os.Getenv("BASE_URL"),
			"SERVER_ADDRESS":    os.Getenv("SERVER_ADDRESS"),
			"FILE_STORAGE_PATH": os.Getenv("FILE_STORAGE_PATH"),
			"DATABASE_DSN":      os.Getenv("DATABASE_DSN"),
			"ENABLE_HTTPS":      os.Getenv("ENABLE_HTTPS"),
			"SSL_PATH":          os.Getenv("SSL_PATH"),
			"CONFIG":            os.Getenv("CONFIG"),
		},
	}

	for _, o := range opts {
		o(configOptions)
	}

	// default:
	// s := config{"http://localhost:8080", "localhost:8080", os.Getenv("HOME") + "/storage.csv", "user=postgres password=postgres host=localhost port=5432 dbname=testdb"}
	s := Config{}

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
		s.dbDSN = v
	}
	if v := configOptions.envVars["SSL_PATH"]; v != "" {
		s.sslPath = v
	}
	if v := configOptions.envVars["CONFIG"]; v != "" {
		configOptions.filePath = v
	}
	if v := configOptions.envVars["ENABLE_HTTPS"]; v != "" {
		tlsModeSet = s.setTLSMode(v)
	}

	if !configOptions.ignoreOsArgs {
		fs := flag.NewFlagSet("myFS", flag.ContinueOnError)
		if !fs.Parsed() {
			var useTLS string

			fs.StringVar(&s.baseURL, "b", s.baseURL, "base for short URLs")
			fs.StringVar(&s.srvAddr, "a", s.srvAddr, "the shortener service address")
			fs.StringVar(&s.storagePath, "f", s.storagePath, "path to a storage file")
			fs.StringVar(&s.dbDSN, "d", s.dbDSN, "db connection path")
			fs.StringVar(&s.sslPath, "p", s.sslPath, "path to folder with .key and .srt files")
			fs.StringVar(&configOptions.filePath, "c", configOptions.filePath, "path to JSON config file")
			fs.StringVar(&configOptions.filePath, "config", configOptions.filePath, "path to JSON config file")
			fs.StringVar(&useTLS, "s", useTLS, "the server will use HTTPS if set to true")

			fs.Parse(configOptions.osArgs)

			tlsModeSet = s.setTLSMode(useTLS)
		}
	}

	if configOptions.filePath == "" {
		// no path to config file is set
		return setDefaults(&s)
	}

	fileData, err := parseFile(configOptions.filePath)
	if err != nil {
		log.Printf("failed to parse config file %v: %v", configOptions.filePath, err)
		return setDefaults(&s)
	}

	if s.baseURL == "" {
		s.baseURL = fileData.BaseUrl
	}
	if s.dbDSN == "" {
		s.dbDSN = fileData.DatabaseDsn
	}
	if s.srvAddr == "" {
		s.srvAddr = fileData.ServerAddress
	}
	if s.storagePath == "" {
		s.storagePath = fileData.FileStoragePath
	}
	if !tlsModeSet {
		s.useTLS = fileData.EnableHttps
	}

	return setDefaults(&s)
}

func (c *Config) setTLSMode(v string) bool {
	if v == "" {
		return false
	}

	use, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("failed to parse bool from ENABLE_HTTPS env var: %v", v)

		return false
	} else {
		c.useTLS = use

		return true
	}
}

type fileStruct struct {
	ServerAddress   string `json:"server_address"`
	BaseUrl         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDsn     string `json:"database_dsn"`
	EnableHttps     bool   `json:"enable_https"`
}

func parseFile(p string) (*fileStruct, error) {
	f, err := os.OpenFile(p, os.O_RDONLY, 0o777)
	if err != nil {
		return nil, err
	}

	data := fileStruct{}

	dec := json.NewDecoder(f)
	err = dec.Decode(&data)
	if err != nil {
		return nil, err
	}

	return &data, nil
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

func (c Config) DBDSN() string {
	return c.dbDSN
}

func (c Config) SslPath() string {
	return c.sslPath
}

func (c Config) UseTLS() bool {
	return c.useTLS
}

func setDefaults(c *Config) *Config {
	if c.srvAddr == "" {
		c.srvAddr = "localhost:8080"
	}
	if c.baseURL == "" {
		c.baseURL = "http://localhost:8080"
	}
	return c
}
