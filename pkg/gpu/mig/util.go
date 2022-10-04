package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"k8s.io/api/core/v1"
	"regexp"
	"strconv"
)

var (
	ResourceRegexp        = regexp.MustCompile(constant.RegexNvidiaMigDevice)
	migDeviceMemoryRegexp = regexp.MustCompile(constant.RegexNvidiaMigFormatMemory)
	numberRegexp          = regexp.MustCompile("\\d+")
)

func IsNvidiaMigDevice(resourceName v1.ResourceName) bool {
	return ResourceRegexp.MatchString(string(resourceName))
}

func ExtractMemoryGBFromMigFormat(migFormatResourceName v1.ResourceName) (int64, error) {
	var err error
	var res int64

	matches := migDeviceMemoryRegexp.FindAllString(string(migFormatResourceName), -1)
	if len(matches) != 1 {
		return res, fmt.Errorf("invalid input string, expected 1 regexp match but found %d", len(matches))
	}
	if res, err = strconv.ParseInt(numberRegexp.FindString(matches[0]), 10, 64); err != nil {
		return res, err
	}

	return res, nil
}
