package utils

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
)

const (
	lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
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

func GetEnvBool(key string, fallback bool) bool {
	value := GetEnv(key, strconv.FormatBool(fallback))
	if v, err := strconv.ParseBool(value); err != nil {
		return fallback
	} else {
		return v
	}
}

func GetEnvOrError(key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	}
	return "", fmt.Errorf("missing env variable %s", key)
}

func RandomStringLowercase(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = lowercaseLetters[rand.Int63()%int64(len(lowercaseLetters))]
	}
	return string(b)
}
