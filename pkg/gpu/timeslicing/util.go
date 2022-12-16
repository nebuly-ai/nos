package timeslicing

import (
	v1 "k8s.io/api/core/v1"
	"strings"
)

func ExtractProfileName(r v1.ResourceName) ProfileName {
	return ProfileName(strings.TrimPrefix(r.String(), profileNamePrefix))
}
