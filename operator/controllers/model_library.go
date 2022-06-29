package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/utils"
)

const (
	// ModelLibraryConfigMapName is the name of the ConfigMap containing the configuration of the model library
	ModelLibraryConfigMapName string = "n8s-model-library-config"
	// ModelLibraryConfigKeyName is the name of the key containing to the config of the model library
	ModelLibraryConfigKeyName string = "config"

	modelInfoFileName string = "model-info.json"

	envModelLibraryAzureClientId     string = "AZ_CLIENT_ID"
	envModelLibraryAzureClientSecret string = "AZ_CLIENT_SECRET"
	envModelLibraryAzureTenantId     string = "AZ_TENANT_ID"
)

type ModelLibraryStorageKind string

const (
	modelLibraryKindAzure ModelLibraryStorageKind = "azure"
	modelLibraryKindS3    ModelLibraryStorageKind = "s3"
	modelLibraryKindMock  ModelLibraryStorageKind = "mock"
)

type ModelDescriptor struct {
}

type ModelLibrary interface {
	// GetCredentials returns a map containing the credentials required for authenticating with the model library, where
	// the keys are the names of the env variables corresponding to each credential
	GetCredentials() (map[string]string, error)
	// GetBaseUri returns the URI pointing to a space within the model library dedicated to the ModelDeployment
	// provided as input argument
	GetBaseUri(modelDeployment *v1alpha1.ModelDeployment) string
	// GetOptimizedModelDescriptorUri return a URI pointing to the file containing the information of the optimized
	// model of the  ModelDeployment provided as input argument
	GetOptimizedModelDescriptorUri(modelDeployment *v1alpha1.ModelDeployment) string
	// FetchOptimizedModelDescriptor returns the ModelDescriptor, if present, of the optimized model produced by
	// the ModelDeployment provided as argument
	FetchOptimizedModelDescriptor(modelDeployment *v1alpha1.ModelDeployment) (*ModelDescriptor, error)
	// GetStorageKind returns the kind of storage used by the model library
	GetStorageKind() ModelLibraryStorageKind
}

type baseModelLibrary struct {
	Uri  string                  `json:"uri"`
	Kind ModelLibraryStorageKind `json:"kind"`
}

func NewModelLibraryFromJson(jsonConfig string) (ModelLibrary, error) {
	var baseModelLibrary baseModelLibrary
	if err := json.Unmarshal([]byte(jsonConfig), &baseModelLibrary); err != nil {
		return nil, err
	}
	switch baseModelLibrary.Kind {
	case modelLibraryKindAzure:
		return newAzureModelLibrary(baseModelLibrary)
	case modelLibraryKindS3:
		return newS3ModelLibrary(baseModelLibrary)
	default:
		return nil, fmt.Errorf("invalid kind %s", baseModelLibrary.Kind)
	}
}

func (b *baseModelLibrary) GetBaseUri(modelDeployment *v1alpha1.ModelDeployment) string {
	return fmt.Sprintf("%s/%s/%s", b.Uri, modelDeployment.Namespace, modelDeployment.Name)
}

func (b *baseModelLibrary) GetOptimizedModelDescriptorUri(modelDeployment *v1alpha1.ModelDeployment) string {
	return fmt.Sprintf("%s/%s", b.GetBaseUri(modelDeployment), modelInfoFileName)
}

func (b *baseModelLibrary) GetStorageKind() ModelLibraryStorageKind {
	return b.Kind
}

// ----------- Azure model library -----------

type azureModelLibrary struct {
	baseModelLibrary
	blobClient *azblob.BlobClient
}

func newAzureModelLibrary(base baseModelLibrary) (*azureModelLibrary, error) {
	var tenantId, clientId, clientSecret string
	var err error

	if clientId, err = utils.GetEnvOrError(envModelLibraryAzureClientId); err != nil {
		return nil, err
	}
	if tenantId, err = utils.GetEnvOrError(envModelLibraryAzureTenantId); err != nil {
		return nil, err
	}
	if clientSecret, err = utils.GetEnvOrError(envModelLibraryAzureClientSecret); err != nil {
		return nil, err
	}

	credential, err := azidentity.NewClientSecretCredential(
		tenantId,
		clientId,
		clientSecret,
		&azidentity.ClientSecretCredentialOptions{},
	)
	if err != nil {
		return nil, err
	}

	client, err := azblob.NewBlobClient(base.Uri, credential, &azblob.ClientOptions{})
	if err != nil {
		return nil, err
	}

	return &azureModelLibrary{baseModelLibrary: base, blobClient: client}, nil
}

func (a azureModelLibrary) GetCredentials() (map[string]string, error) {
	var tenantId, clientId, clientSecret string
	var err error

	if clientId, err = utils.GetEnvOrError(envModelLibraryAzureClientId); err != nil {
		return nil, err
	}
	if tenantId, err = utils.GetEnvOrError(envModelLibraryAzureTenantId); err != nil {
		return nil, err
	}
	if clientSecret, err = utils.GetEnvOrError(envModelLibraryAzureClientSecret); err != nil {
		return nil, err
	}

	return map[string]string{
		envModelLibraryAzureTenantId:     tenantId,
		envModelLibraryAzureClientId:     clientId,
		envModelLibraryAzureClientSecret: clientSecret,
	}, nil
}

func (a azureModelLibrary) FetchOptimizedModelDescriptor(modelDeployment *v1alpha1.ModelDeployment) (*ModelDescriptor, error) {
	//TODO implement me
	panic("implement me")
}

// ----------- S3 model library -----------

type s3ModelLibrary struct {
	baseModelLibrary
}

func newS3ModelLibrary(base baseModelLibrary) (*s3ModelLibrary, error) {
	return &s3ModelLibrary{baseModelLibrary: base}, nil
}

func (s s3ModelLibrary) GetCredentials() (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}

func (s s3ModelLibrary) FetchOptimizedModelDescriptor(modelDeployment *v1alpha1.ModelDeployment) (*ModelDescriptor, error) {
	//TODO implement me
	panic("implement me")
}

// ----------- Mock model library -----------

type mockModelLibrary struct {
	baseModelLibrary
	returnedCredentials  map[string]string
	returnedModelInfoUri string
	returnedModelInfo    *ModelDescriptor
	returnedBaseUri      string
}

func newMockModelLibrary(base baseModelLibrary) *mockModelLibrary {
	return &mockModelLibrary{
		baseModelLibrary:    base,
		returnedCredentials: make(map[string]string, 0),
	}
}

func (m mockModelLibrary) GetCredentials() (map[string]string, error) {
	return m.returnedCredentials, nil
}

func (m mockModelLibrary) GetBaseUri(modelDeployment *v1alpha1.ModelDeployment) string {
	return m.returnedBaseUri
}

func (m mockModelLibrary) GetOptimizedModelDescriptorUri(modelDeployment *v1alpha1.ModelDeployment) string {
	return m.returnedModelInfoUri
}

func (m mockModelLibrary) FetchOptimizedModelDescriptor(modelDeployment *v1alpha1.ModelDeployment) (*ModelDescriptor, error) {
	return m.returnedModelInfo, nil
}
