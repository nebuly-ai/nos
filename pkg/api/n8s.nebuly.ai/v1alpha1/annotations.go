package v1alpha1

import (
	"fmt"
)

const (
	AnnotationGPUSpecPrefix       = "n8s.nebuly.ai/spec-gpu"
	AnnotationGPUMigSpecFormat    = "n8s.nebuly.ai/spec-gpu-%d-%s"
	AnnotationGPUStatusPrefix     = "n8s.nebuly.ai/status-gpu"
	AnnotationGPUStatusFreeSuffix = "free"
	AnnotationGPUStatusUsedSuffix = "used"
)

var (
	AnnotationUsedMigStatusFormat = fmt.Sprintf("%s-%%d-%%s-%s", AnnotationGPUStatusPrefix, AnnotationGPUStatusUsedSuffix)
	AnnotationFreeMigStatusFormat = fmt.Sprintf("%s-%%d-%%s-%s", AnnotationGPUStatusPrefix, AnnotationGPUStatusFreeSuffix)
)
