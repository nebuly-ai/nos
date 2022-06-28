package controllers

const (
	EventInternalError              = "InternalError"
	EventModelOptimizationFailed    = "ModelOptimizationFailed"
	EventModelOptimizationCompleted = "ModelOptimizationCompleted"

	LabelCreatedBy = "app.kubernetes.io/created-by"

	EnvSkipControllerTests = "SKIP_CONTROLLER_TESTS"
)
