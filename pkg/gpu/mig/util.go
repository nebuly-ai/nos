package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"k8s.io/api/core/v1"
	"regexp"
	"strconv"
	"strings"
)

var (
	resourceRegexp        = regexp.MustCompile(constant.RegexNvidiaMigResource)
	migDeviceMemoryRegexp = regexp.MustCompile(constant.RegexNvidiaMigFormatMemory)
	numberRegexp          = regexp.MustCompile(`\d+`)
)

func IsNvidiaMigDevice(resourceName v1.ResourceName) bool {
	return resourceRegexp.MatchString(string(resourceName))
}

// ExtractMigProfile extracts the name of the MIG profile from the provided resource name, and returns an error
// if the resource name is not a valid NVIDIA MIG resource.
//
// Example:
//
//	nvidia.com/mig-1g.10gb => 1g.10gb
func ExtractMigProfile(migFormatResourceName v1.ResourceName) (string, error) {
	if isMigResource := resourceRegexp.MatchString(string(migFormatResourceName)); !isMigResource {
		return "", fmt.Errorf("invalid input string, required format is %s", resourceRegexp.String())
	}
	return strings.TrimPrefix(string(migFormatResourceName), "nvidia.com/mig-"), nil
}

func ExtractMemoryGBFromMigFormat(migFormatResourceName v1.ResourceName) (int64, error) {
	var err error
	var res int64

	if isMigResource := resourceRegexp.MatchString(string(migFormatResourceName)); !isMigResource {
		return res, fmt.Errorf("invalid input string, required format is %s", resourceRegexp.String())
	}

	matches := migDeviceMemoryRegexp.FindAllString(string(migFormatResourceName), -1)
	if len(matches) != 1 {
		return res, fmt.Errorf("invalid input string, expected 1 regexp match but found %d", len(matches))
	}
	if res, err = strconv.ParseInt(numberRegexp.FindString(matches[0]), 10, 64); err != nil {
		return res, err
	}

	return res, nil
}
