package mig

import (
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"regexp"
	"strconv"
	"strings"
)

const (
	profile1g6gb  ProfileName = "1g.6gb"
	profile2g12gb ProfileName = "2g.12gb"
	profile4g24gb ProfileName = "4g.24gb"

	profile1g5gb  ProfileName = "1g.5gb"
	profile2g10gb ProfileName = "2g.10gb"
	profile3g20gb ProfileName = "3g.20gb"
	profile4g20gb ProfileName = "4g.20gb"
	profile7g40gb ProfileName = "7g.40gb"

	profile1g10gb ProfileName = "1g.10gb"
	profile2g20gb ProfileName = "2g.20gb"
	profile3g40gb ProfileName = "3g.40gb"
	profile4g40gb ProfileName = "4g.40gb"
	profile7g80gb ProfileName = "7g.80gb"
)

var (
	migProfileRegex = regexp.MustCompile(constant.RegexNvidiaMigProfile)
	migGiRegex      = regexp.MustCompile(`\d+g`)
	migMemoryRegex  = regexp.MustCompile(`\d+gb`)
)

type ProfileName string

func (p ProfileName) isValid() bool {
	return migProfileRegex.MatchString(string(p))
}

func (p ProfileName) AsString() string {
	return string(p)
}

func (p ProfileName) getMemorySlices() int {
	asString := migMemoryRegex.FindString(string(p))
	asString = strings.TrimSuffix(asString, "gb")
	asInt, _ := strconv.Atoi(asString)
	return asInt
}

func (p ProfileName) getGiSlices() int {
	asString := migGiRegex.FindString(string(p))
	asString = strings.TrimSuffix(asString, "g")
	asInt, _ := strconv.Atoi(asString)
	return asInt
}

type Profile struct {
	GpuIndex int
	Name     ProfileName
}
