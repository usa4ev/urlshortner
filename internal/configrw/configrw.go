package configrw

import (
	"flag"
	"os"
)

var (
	baseURL     = "http://localhost:8080/"
	srvAddr     = "localhost:8080"
	storagePath = "$HOME"
)

func ReadBaseURL() string {
	if baseURL != "" {
		return baseURL
	}

	baseURL = os.Getenv("BASE_URL")

	return baseURL
}

func ReadSrvAddr() string {
	if srvAddr != "" {
		return srvAddr
	}

	srvAddr = os.Getenv("BASE_URL")

	return srvAddr
}

func ReadStoragePath() string {
	if storagePath != "" {
		return storagePath
	}

	storagePath = os.Getenv("BASE_URL")

	return storagePath
}

func ParseFlags() {
	flag.StringVar(&baseURL, "b", baseURL, "base for short URLs")
	flag.StringVar(&srvAddr, "a", srvAddr, "address of URL the shortener service")
	flag.StringVar(&storagePath, "f", storagePath, "path to a storage file")
	flag.Parse()
}
