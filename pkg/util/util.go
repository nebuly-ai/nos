package util

import (
	"fmt"
	"golang.org/x/exp/constraints"
	"k8s.io/apimachinery/pkg/types"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

const (
	lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
)

func BoolAddr(b bool) *bool {
	var boolVar = b
	return &boolVar
}

func StringAddr(s string) *string {
	var stringVar = s
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

func GetNamespacedName(object client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}
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

func Min[K constraints.Ordered](v1 K, v2 K) K {
	if v1 < v2 {
		return v1
	}
	return v2
}

func Max[K constraints.Ordered](v1 K, v2 K) K {
	if v1 > v2 {
		return v1
	}
	return v2
}

func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func InSlice[K comparable](item K, slice []K) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

// LocalEndpoint returns the full path to a unix socket at the given endpoint
func LocalEndpoint(path, file string) (string, error) {
	u := url.URL{
		Scheme: "unix",
		Path:   path,
	}
	return filepath.Join(u.String(), file+".sock"), nil
}
