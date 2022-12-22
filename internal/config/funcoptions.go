package config

import "os"

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

func setConfigOptions(opts []configOption) *configOptions {
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

	if v := configOptions.envVars["CONFIG"]; v != "" {
		configOptions.filePath = v
	}
	return configOptions
}

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
