package components

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/utils"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
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
	ModelLibraryKindAzure ModelLibraryStorageKind = "azure"
	ModelLibraryKindS3    ModelLibraryStorageKind = "s3"
	ModelLibraryKindMock  ModelLibraryStorageKind = "mock"
)

const (
	modelLibraryRequestsTimeout = 1 * time.Second
)

type ModelDescriptor struct {
	ModelUri string `json:"model_uri"`
}

func NewModelDescriptorFromJson(jsonBytes []byte) (*ModelDescriptor, error) {
	var modelDescriptor = new(ModelDescriptor)
	if err := json.Unmarshal(jsonBytes, modelDescriptor); err != nil {
		return nil, err
	}
	return modelDescriptor, nil
}

func (m *ModelDescriptor) AsMap() map[string]string {
	// TODO: more efficient implementation of Struct -> Map conversion, handle errors
	var configMapData = new(map[string]string)
	j, _ := json.Marshal(m)
	_ = json.Unmarshal(j, configMapData)
	return *configMapData
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
	FetchOptimizedModelDescriptor(ctx context.Context, modelDeployment *v1alpha1.ModelDeployment) (*ModelDescriptor, error)
	// GetStorageKind returns the kind of storage used by the model library
	GetStorageKind() ModelLibraryStorageKind
}

type BaseModelLibrary struct {
	Uri  string                  `json:"uri"`
	Kind ModelLibraryStorageKind `json:"kind"`
}

func NewModelLibraryFromJson(jsonConfig string) (ModelLibrary, error) {
	var baseModelLibrary BaseModelLibrary
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

func (b *BaseModelLibrary) GetBaseUri(modelDeployment *v1alpha1.ModelDeployment) string {
	return fmt.Sprintf("%s/%s/%s", b.Uri, modelDeployment.Namespace, modelDeployment.Name)
}

func (b *BaseModelLibrary) GetOptimizedModelDescriptorUri(modelDeployment *v1alpha1.ModelDeployment) string {
	return fmt.Sprintf("%s/%s", b.GetBaseUri(modelDeployment), modelInfoFileName)
}

func (b *BaseModelLibrary) GetStorageKind() ModelLibraryStorageKind {
	return b.Kind
}

// ----------- Azure model library -----------

type azureModelLibrary struct {
	BaseModelLibrary
	credential azcore.TokenCredential
}

func newAzureModelLibrary(base BaseModelLibrary) (*azureModelLibrary, error) {
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

	return &azureModelLibrary{BaseModelLibrary: base, credential: credential}, nil
}
func (a *azureModelLibrary) newBlobClient(uri string) (*azblob.BlobClient, error) {
	blobOptions := &azblob.ClientOptions{
		Retry: policy.RetryOptions{
			MaxRetries: 0,
			TryTimeout: modelLibraryRequestsTimeout,
		},
	}
	return azblob.NewBlobClient(uri, a.credential, blobOptions)
}

func (a *azureModelLibrary) GetCredentials() (map[string]string, error) {
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

func (a *azureModelLibrary) FetchOptimizedModelDescriptor(ctx context.Context, modelDeployment *v1alpha1.ModelDeployment) (*ModelDescriptor, error) {
	logger := log.FromContext(ctx)

	uri := a.GetOptimizedModelDescriptorUri(modelDeployment)
	logger.V(1).Info("downloading optimized model descriptor", "uri", uri)

	client, err := a.newBlobClient(uri)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize azure blob client")
	}
	resp, err := client.Download(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error downloading model optimizer descriptor from Azure Blob")
	}

	reader := resp.Body(&azblob.RetryReaderOptions{})
	defer reader.Close()

	modelDescriptorBytes := new(bytes.Buffer)
	_, err = modelDescriptorBytes.ReadFrom(reader)
	if err != nil {
		return nil, errors.Wrap(err, "error reading model optimizer descriptor from Azure Blob")
	}

	return NewModelDescriptorFromJson(modelDescriptorBytes.Bytes())
}

// ----------- S3 model library -----------

type s3ModelLibrary struct {
	BaseModelLibrary
}

func newS3ModelLibrary(base BaseModelLibrary) (*s3ModelLibrary, error) {
	return &s3ModelLibrary{BaseModelLibrary: base}, nil
}

func (s *s3ModelLibrary) GetCredentials() (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}

func (s *s3ModelLibrary) FetchOptimizedModelDescriptor(ctx context.Context, modelDeployment *v1alpha1.ModelDeployment) (*ModelDescriptor, error) {
	//TODO implement me
	panic("implement me")
}

// ----------- Mock model library -----------

type mockModelLibrary struct {
	BaseModelLibrary
	returnedCredentials     map[string]string
	returnedModelInfoUri    string
	returnedModelDescriptor *ModelDescriptor
	returnedBaseUri         string
}

func NewMockModelLibrary(base BaseModelLibrary, modelDescriptor *ModelDescriptor) *mockModelLibrary {
	return &mockModelLibrary{
		BaseModelLibrary:        base,
		returnedCredentials:     make(map[string]string, 0),
		returnedModelDescriptor: modelDescriptor,
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

func (m mockModelLibrary) FetchOptimizedModelDescriptor(ctx context.Context, modelDeployment *v1alpha1.ModelDeployment) (*ModelDescriptor, error) {
	return m.returnedModelDescriptor, nil
}
