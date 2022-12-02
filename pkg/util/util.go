/*
 * Copyright 2022 Nebuly.ai
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/constraints"
	"hash/fnv"
	"k8s.io/apimachinery/pkg/types"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
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

type HasNamespacedName interface {
	GetName() string
	GetNamespace() string
}

func GetNamespacedName(object HasNamespacedName) types.NamespacedName {
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

func CopyMap[K comparable, V any](m map[K]V) map[K]V {
	var res = make(map[K]V, len(m))
	for k, v := range m {
		res[k] = v
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

func Filter[K any](slice []K, filter func(k K) bool) []K {
	var res = make([]K, 0)
	for _, k := range slice {
		if filter(k) {
			res = append(res, k)
		}
	}
	return res
}

func UnorderedEqual[K any](first []K, second []K) bool {
	firstLen := len(first)
	secondLen := len(second)
	if firstLen != secondLen {
		return false
	}

	visited := make([]bool, firstLen)

	for i := 0; i < firstLen; i++ {
		found := false
		element := first[i]
		for j := 0; j < secondLen; j++ {
			if visited[j] {
				continue
			}
			if cmp.Equal(element, second[j]) {
				visited[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// LocalEndpoint returns the full path to a unix socket at the given endpoint
func LocalEndpoint(path, file string) (string, error) {
	u := url.URL{
		Scheme: "unix",
		Path:   path,
	}
	return filepath.Join(u.String(), file+".sock"), nil
}

func HashFnv32a(str string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(str))
	return string(h.Sum(nil))
}
