package configrw

import (
	"fmt"
	"os"
)

func BaseURL() string {
	if v, ok := os.LookupEnv("BASE_URL"); ok {
		return v
	}
	fmt.Println("failed to read environment var BASE_URL")

	return "http://localhost:8080/"
}

func SrvAddr() string {
	if v, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		return v
	}
	fmt.Println("failed to read environment var SERVER_ADDRESS")

	return "localhost:8080"
}
