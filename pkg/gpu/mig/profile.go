package mig

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"regexp"
)

var (
	migProfileRegex = regexp.MustCompile(constant.RegexNvidiaMigProfile)
)

type ProfileName string

func (p ProfileName) isValid() bool {
	return migProfileRegex.MatchString(string(p))
}

func (p ProfileName) AsString() string {
	return string(p)
}

func getMemorySlices() uint8 {
	return 0
}

func getGiSlices() uint8 {
	return 0
}

type Profile struct {
	GpuIndex int
	Name     ProfileName
}
