package util

import (
	"fmt"
	"math/rand"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func StringAddr(s string) *string {
	var stringVar string
	stringVar = s
	return &stringVar
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

func GetNamespacedName(object client.Object) string {
	return fmt.Sprintf("%s/%s", object.GetNamespace(), object.GetName())
}

type empty struct {
}

func GetKeys[K comparable, V any](maps ...map[K]V) []K {
	var set = make(map[K]empty)
	for _, m := range maps {
		for k := range m {
			set[k] = empty{}
		}
	}
	var res = make([]K, len(set))
	var i int
	for k := range set {
		res[i] = k
		i++
	}
	return res
}
