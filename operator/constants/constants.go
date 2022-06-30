package constants

// ExistenceCheckResult represents the result of an existence check on a certain resource
type ExistenceCheckResult string

const (
	// ExistenceCheckExists indicates that the resource exists, and it is up-to-date with the spec
	ExistenceCheckExists ExistenceCheckResult = "exists"
	// ExistenceCheckUpdate indicates that the resource exists, but needs to be updated
	ExistenceCheckUpdate ExistenceCheckResult = "update"
	// ExistenceCheckCreate indicates that the resource does not exist yet and needs to be created
	ExistenceCheckCreate ExistenceCheckResult = "create"
	// ExistenceCheckError indicates that an error occurred when checking the existence of the resource
	ExistenceCheckError ExistenceCheckResult = "error"
)

const (
	EventInternalError           = "InternalError"
	EventModelOptimizationFailed = "ModelOptimizationFailed"
	EventModelDeploymentUpdated  = "ModelDeploymentUpdated"

	LabelCreatedBy          = "app.kubernetes.io/created-by"
	LabelOptimizationTarget = "n8s.nebuly.ai/optimization-target"
	LabelModelDeployment    = "n8s.nebuly.ai/model-deployment"
	LabelOptimizationJob    = "n8s.nebuly.ai/optimization-job"

	AnnotationSourceModelUri = "n8s.nebuly.ai/source-model-uri"

	EnvSkipControllerTests = "SKIP_CONTROLLER_TESTS"

	ControllerManagerName = "n8s-controller-manager"

	// ModelDeploymentControllerName is the name of the controller of ModelDeployment kind
	ModelDeploymentControllerName = "modeldeployment-controller"

	// OptimizationJobNamePrefix is the prefix used for the auto-generated names of the model optimization jobs
	OptimizationJobNamePrefix = "optimization-"
	// ModelDescriptorNamePrefix is the prefix used for the auto-generated names of the model descriptor config maps
	ModelDescriptorNamePrefix = "model-descriptor-"
)
