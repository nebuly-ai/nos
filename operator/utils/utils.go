package utils

import "os"

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
