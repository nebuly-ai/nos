package controllers

const (
	// ModelLibraryConfigMapName is the name of the ConfigMap containing the configuration of the model library
	ModelLibraryConfigMapName = "n8s-model-library-config"
	// ModelLibraryCredentialsSecretName is the name of the Secret containing the credentials for authenticating with
	// the model library
	ModelLibraryCredentialsSecretName = "n8s-model-library-credentials"
	// ModelLibraryConfigKeyName is the name of the key containing to the config of the model library
	ModelLibraryConfigKeyName = "config"

	EventInternalError              = "InternalError"
	EventModelOptimizationFailed    = "ModelOptimizationFailed"
	EventModelOptimizationCompleted = "ModelOptimizationCompleted"

	LabelCreatedBy = "app.kubernetes.io/created-by"
)
