package constants

const (
	EventInternalError           = "InternalError"
	EventModelOptimizationFailed = "ModelOptimizationFailed"
	EventModelDeploymentUpdated  = "ModelDeploymentUpdated"

	LabelCreatedBy          = "app.kubernetes.io/created-by"
	LabelOptimizationTarget = "n8s.nebuly.ai/optimization-target"

	AnnotationSourceModelUri = "n8s.nebuly.ai/source-model-uri"

	EnvSkipControllerTests = "SKIP_CONTROLLER_TESTS"

	ControllerManagerName = "n8s-controller-manager"
)
