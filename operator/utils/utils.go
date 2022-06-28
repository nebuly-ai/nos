package utils

import (
	"fmt"
	"os"
)

func BoolAddr(b bool) *bool {
	var boolVar bool
	boolVar = b
	return &boolVar
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvOrError(key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	}
	return "", fmt.Errorf("missing env variable %s", key)
}
