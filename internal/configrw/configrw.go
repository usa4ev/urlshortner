package configrw

import (
	"fmt"
	"os"
)

func ReadBaseURL() string {
	if v, ok := os.LookupEnv("BASE_URL"); ok {
		return v
	}
	fmt.Println("failed to read environment var BASE_URL")

	return "http://localhost:8080/"
}

func ReadSrvAddr() string {
	if v, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		return v
	}
	fmt.Println("failed to read environment var SERVER_ADDRESS")

	return "localhost:8080"
}

func ReadStoragePath() string {
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		return v
	}
	fmt.Println("failed to read environment var FILE_STORAGE_PATH")

	return ""
}
