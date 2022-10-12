package v1alpha1

// Annotations
const (
	AnnotationGPUSpecPrefix    = "n8s.nebuly.ai/spec-gpu"
	AnnotationGPUMigSpecFormat = "n8s.nebuly.ai/spec-gpu-%d-%s"

	AnnotationGPUStatusPrefix     = "n8s.nebuly.ai/status-gpu"
	AnnotationUsedMigStatusFormat = "n8s.nebuly.ai/status-gpu-%d-%s-used"
	AnnotationFreeMigStatusFormat = "n8s.nebuly.ai/status-gpu-%d-%s-free"
)
