package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
)

const (
	priorityFile = iota
	priorityEnvVars
	priorityOsArgs
	topPriority
)

type Config struct {
	baseURL       string
	srvAddr       string
	storagePath   string
	dbDSN         string
	sslPath       string
	trustedSubnet string
	useTLS        bool
	useGRPC       bool
	grpcModeSet   bool
}

func New(opts ...configOption) *Config {
	configOptions := setConfigOptions(opts)

	configs := make([]*pConfig, topPriority)

	configs[priorityEnvVars] = fromEnv(configOptions.envVars)

	if !configOptions.ignoreOsArgs {
		configs[priorityOsArgs] = fromArgs(configOptions.osArgs, &configOptions.filePath)
	}

	if !configOptions.ignoreCfgFile && configOptions.filePath != "" {
		configs[priorityFile] = fromFile(configOptions.filePath)
	}

	return fillCfg(configs)
}

// fillCfg fills Config from passed collection of pConfig
// from different sources considering priority.
func fillCfg(configs []*pConfig) *Config {
	cfg := Config{}

	for _, pCfg := range configs {
		if pCfg == nil {
			continue
		}

		if pCfg.srvAddr != "" {
			cfg.srvAddr = pCfg.srvAddr
		}
		if pCfg.baseURL != "" {
			cfg.baseURL = pCfg.baseURL
		}
		if pCfg.storagePath != "" {
			cfg.storagePath = pCfg.storagePath
		}
		if pCfg.dbDSN != "" {
			cfg.dbDSN = pCfg.dbDSN
		}
		if pCfg.sslPath != "" {
			cfg.sslPath = pCfg.sslPath
		}
		if pCfg.trustedSubnet != "" {
			cfg.trustedSubnet = pCfg.trustedSubnet
		}
		if pCfg.tlsModeSet {
			cfg.useTLS = pCfg.useTLS
		}

	}

	return cfg.setDefaults()
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

func (c Config) TrustedSubnet() string {
	return c.trustedSubnet
}

func (c Config) GRPC() bool {
	return c.useGRPC
}

func (c *Config) setDefaults() *Config {
	if c.srvAddr == "" {
		c.srvAddr = "localhost:8080"
	}
	if c.baseURL == "" {
		c.baseURL = "http://localhost:8080"
	}
	if c.sslPath == "" {
		c.sslPath = "./etc/ssl"
	}

	return c
}

// pConfig is a temporary Config with service fields
type pConfig struct {
	Config
	tlsModeSet bool // marks if useTLS param is set
}

func newpConfig() pConfig {
	return pConfig{
		Config:     Config{},
		tlsModeSet: false,
	}
}

func fromEnv(envVars map[string]string) *pConfig {
	pc := newpConfig()

	if v := envVars["BASE_URL"]; v != "" {
		pc.baseURL = v
	}
	if v := envVars["SERVER_ADDRESS"]; v != "" {
		pc.srvAddr = v
	}
	if v := envVars["FILE_STORAGE_PATH"]; v != "" {
		pc.storagePath = v
	}
	if v := envVars["DATABASE_DSN"]; v != "" {
		pc.dbDSN = v
	}
	if v := envVars["TRUSTED_SUBNET"]; v != "" {
		pc.trustedSubnet = v
	}
	if v := envVars["SSL_PATH"]; v != "" {
		pc.sslPath = v
	}
	if v := envVars["ENABLE_HTTPS"]; v != "" {
		pc.setTLSMode(v)
	}
	if v := envVars["USE_GRPC"]; v != "" {
		pc.setGrpcMode(v)
	}

	return &pc
}

func fromArgs(osArgs []string, filePath *string) *pConfig {
	pc := newpConfig()
	fs := flag.NewFlagSet("myFS", flag.ContinueOnError)
	if !fs.Parsed() {
		var useTLS, useGRPC string

		fs.StringVar(&pc.baseURL, "b", "", "base for short URLs")
		fs.StringVar(&pc.srvAddr, "a", "", "the shortener service address")
		fs.StringVar(&pc.storagePath, "f", "", "path to a storage file")
		fs.StringVar(&pc.dbDSN, "d", "", "db connection path")
		fs.StringVar(&pc.trustedSubnet, "t", "", "trusted subnet subnet to accept calls without authentication")
		fs.StringVar(&pc.sslPath, "p", "", "path to folder with .key and .srt files")
		fs.StringVar(filePath, "c", *filePath, "path to JSON config file")
		fs.StringVar(filePath, "config", *filePath, "path to JSON config file")
		fs.StringVar(&useTLS, "s", useTLS, "the server will use HTTPS if set to true")
		fs.StringVar(&useGRPC, "r", useGRPC, "the server will start as gRPC-server")

		fs.Parse(osArgs)

		pc.setTLSMode(useTLS)
		pc.setGrpcMode(useGRPC)
	}

	return &pc
}

func fromFile(filePath string) *pConfig {
	pc := newpConfig()
	fileData, err := parseFile(filePath)

	if err != nil {
		log.Printf("failed to parse config file %v: %v", filePath, err)
		return &pc
	}

	pc.baseURL = fileData.BaseUrl
	pc.dbDSN = fileData.DatabaseDsn
	pc.srvAddr = fileData.ServerAddress
	pc.storagePath = fileData.FileStoragePath
	pc.useTLS = fileData.EnableHttps
	pc.sslPath = fileData.SslPath
	pc.trustedSubnet = fileData.TrustedSubnet
	pc.tlsModeSet = true
	pc.useGRPC = fileData.UseGrpc
	pc.grpcModeSet = true

	return &pc
}

type fileStruct struct {
	ServerAddress   string `json:"server_address"`
	BaseUrl         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDsn     string `json:"database_dsn"`
	TrustedSubnet   string `json:"trusted_subnet"`
	EnableHttps     bool   `json:"enable_https"`
	SslPath         string `json:"ssl_path"`
	UseGrpc         bool   `json:"use_grpc"`
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

func (pc *pConfig) setTLSMode(v string) {
	if v == "" {
		return
	}

	use, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("failed to parse bool from ENABLE_HTTPS env var: %v", v)

		return
	} else {
		pc.useTLS = use
		pc.tlsModeSet = true
		return
	}
}

func (pc *pConfig) setGrpcMode(v string) {
	if v == "" {
		return
	}

	use, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("failed to parse bool of grpc mode flag: %v", v)

		return
	} else {
		pc.useGRPC = use
		pc.grpcModeSet = true
		return
	}
}
