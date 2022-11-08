package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	"regexp"
	"strconv"
	"strings"
)

const (
	Profile1g6gb  ProfileName = "1g.6gb"
	Profile2g12gb ProfileName = "2g.12gb"
	Profile4g24gb ProfileName = "4g.24gb"

	Profile1g5gb  ProfileName = "1g.5gb"
	Profile2g10gb ProfileName = "2g.10gb"
	Profile3g20gb ProfileName = "3g.20gb"
	Profile4g20gb ProfileName = "4g.20gb"
	Profile7g40gb ProfileName = "7g.40gb"

	Profile1g10gb ProfileName = "1g.10gb"
	Profile2g20gb ProfileName = "2g.20gb"
	Profile3g40gb ProfileName = "3g.40gb"
	Profile4g40gb ProfileName = "4g.40gb"
	Profile7g80gb ProfileName = "7g.80gb"
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

func (p ProfileName) AsResourceName() v1.ResourceName {
	resourceNameStr := fmt.Sprintf("%s%s", constant.NvidiaMigResourcePrefix, p)
	return v1.ResourceName(resourceNameStr)
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

type ProfileList []Profile

func (p ProfileList) GroupByGPU() map[int]ProfileList {
	return nil // todo
}
