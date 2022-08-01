package configrw

import (
	"flag"
	"os"
)

var (
	baseURL     = "http://localhost:8080"
	srvAddr     = "localhost:8080"
	storagePath = "$HOME"
)

func ReadBaseURL() string {
	return baseURL
}

func ReadSrvAddr() string {
	return srvAddr
}

func ReadStoragePath() string {
	return storagePath
}

func ParseFlags() {

	// setting up default values first
	envVars := map[string]*string{
		"BASE_URL":          &baseURL,
		"SERVER_ADDRESS":    &srvAddr,
		"FILE_STORAGE_PATH": &storagePath,
	}
	for key, envVar := range envVars {
		if v, ok := os.LookupEnv(key); ok {
			*envVar = v
		}
	}

	flag.StringVar(&baseURL, "b", baseURL, "base for short URLs")
	flag.StringVar(&srvAddr, "a", srvAddr, "address of URL the shortener service")
	flag.StringVar(&storagePath, "f", storagePath, "path to a storage file")
	flag.Parse()
}
