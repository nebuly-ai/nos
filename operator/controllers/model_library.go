package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/utils"
)

type ModelLibraryKind string

const (
	ModelLibraryKindAzure ModelLibraryKind = "azure"
	ModelLibraryKindS3    ModelLibraryKind = "s3"

	EnvModelLibraryAzureClientId     string = "AZ_CLIENT_ID"
	EnvModelLibraryAzureClientSecret string = "AZ_CLIENT_SECRET"
	EnvModelLibraryAzureTenantId     string = "AZ_TENANT_ID"
)

type ModelInfo struct {
}

type ModelLibrary interface {
	GetBaseUri() string
	GetCredentials() (map[string]string, error)
	GetModelInfoUri(modelDeployment *v1alpha1.ModelDeployment) string
	FetchModelInfo(modelDeployment *v1alpha1.ModelDeployment) (*ModelInfo, error)
}

type baseModelLibrary struct {
	Uri  string           `json:"uri"`
	Kind ModelLibraryKind `json:"kind"`
}

func NewModelLibraryFromJson(jsonConfig string) (ModelLibrary, error) {
	var baseModelLibrary baseModelLibrary
	if err := json.Unmarshal([]byte(jsonConfig), &baseModelLibrary); err != nil {
		return nil, err
	}
	switch baseModelLibrary.Kind {
	case ModelLibraryKindAzure:
		return newAzureModelLibrary(baseModelLibrary)
	case ModelLibraryKindS3:
		return newS3ModelLibrary(baseModelLibrary)
	default:
		return nil, fmt.Errorf("invalid kind %s", baseModelLibrary.Kind)
	}
}

// ----------- Azure model library -----------

type azureModelLibrary struct {
	baseModelLibrary
	blobClient *azblob.BlobClient
}

func newAzureModelLibrary(base baseModelLibrary) (*azureModelLibrary, error) {
	var tenantId, clientId, clientSecret string
	var err error

	if clientId, err = utils.GetEnvOrError(EnvModelLibraryAzureClientId); err != nil {
		return nil, err
	}
	if tenantId, err = utils.GetEnvOrError(EnvModelLibraryAzureTenantId); err != nil {
		return nil, err
	}
	if clientSecret, err = utils.GetEnvOrError(EnvModelLibraryAzureClientSecret); err != nil {
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

func (a azureModelLibrary) GetBaseUri() string {
	//TODO implement me
	panic("implement me")
}

func (a azureModelLibrary) GetCredentials() (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}

func (a azureModelLibrary) GetModelInfoUri(modelDeployment *v1alpha1.ModelDeployment) string {
	//TODO implement me
	panic("implement me")
}

func (a azureModelLibrary) FetchModelInfo(modelDeployment *v1alpha1.ModelDeployment) (*ModelInfo, error) {
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

func (s s3ModelLibrary) GetBaseUri() string {
	//TODO implement me
	panic("implement me")
}

func (s s3ModelLibrary) GetCredentials() (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}

func (s s3ModelLibrary) GetModelInfoUri(modelDeployment *v1alpha1.ModelDeployment) string {
	//TODO implement me
	panic("implement me")
}

func (s s3ModelLibrary) FetchModelInfo(modelDeployment *v1alpha1.ModelDeployment) (*ModelInfo, error) {
	//TODO implement me
	panic("implement me")
}
