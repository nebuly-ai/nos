package components

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"os"
)

func newModelLibraryJson(uri, kind string) string {
	return fmt.Sprintf(
		"{\"uri\": \"%s\", \"kind\": \"%s\" }",
		uri,
		kind,
	)
}

var _ = Describe("New model library from json config", func() {
	const (
		azureClientSecret = "client-secret"
		azureClientId     = "client-id"
		azureTenantId     = "tenant-id"
	)

	BeforeEach(func() {
		Expect(os.Setenv(envModelLibraryAzureClientSecret, azureClientSecret)).To(Succeed())
		Expect(os.Setenv(envModelLibraryAzureClientId, azureClientId)).To(Succeed())
		Expect(os.Setenv(envModelLibraryAzureTenantId, azureTenantId)).To(Succeed())
	})

	When("JSON config is empty", func() {
		It("Should return an error", func() {
			Expect(NewModelLibraryFromJson("")).Error().To(HaveOccurred())
		})
	})

	When("JSON config is malformed", func() {
		It("Should return an error", func() {
			Expect(NewModelLibraryFromJson("malformed")).Error().To(HaveOccurred())
		})
	})

	When("JSON has missing field", func() {
		It("Should return an error", func() {
			Expect(NewModelLibraryFromJson("{\"foo\": \"bar\"")).Error().To(HaveOccurred())
		})
	})

	When("JSON has extra fields", func() {
		uri := "https://foo.bar"
		kind := ModelLibraryKindAzure
		json := fmt.Sprintf(
			"{\"uri\": \"%s\", \"kind\": \"%s\", \"foo\": \"bar\"}",
			uri,
			kind,
		)
		It("Should ignore extra fields", func() {
			modelLibrary, err := NewModelLibraryFromJson(json)
			Expect(err).ToNot(HaveOccurred())
			Expect(modelLibrary).ToNot(BeNil())
			var azureModelLibraryVar *azureModelLibrary
			Expect(modelLibrary).To(BeAssignableToTypeOf(azureModelLibraryVar))
		})
	})

	When("Model library kind is invalid", func() {
		json := newModelLibraryJson("https://foo.bar", "invalid")
		It("Should return an error", func() {
			Expect(NewModelLibraryFromJson(json)).Error().To(HaveOccurred())
		})
	})

	When("Any required env variable is missing ", func() {
		json := newModelLibraryJson("https://foo.bar", string(ModelLibraryKindAzure))
		It("Should return an error", func() {
			Expect(os.Unsetenv(envModelLibraryAzureTenantId)).To(Succeed())
			Expect(NewModelLibraryFromJson(json)).Error().To(HaveOccurred())
		})
	})

	DescribeTable("Model library kinds from JSON",
		func(kind ModelLibraryStorageKind, expectedModelLibraryType interface{}) {
			json := newModelLibraryJson("https://foo.bar", string(kind))
			modelLibary, err := NewModelLibraryFromJson(json)
			Expect(err).ToNot(HaveOccurred())
			Expect(modelLibary).To(BeAssignableToTypeOf(expectedModelLibraryType))
		},
		Entry("Kind Azure", ModelLibraryKindAzure, &azureModelLibrary{}),
		Entry("Kind S3", ModelLibraryKindS3, &s3ModelLibrary{}),
	)
})

var _ = Describe("GetCredentials", func() {
	When("Model Library kind is Azure", func() {
		const (
			azureClientSecret = "client-secret"
			azureClientId     = "client-id"
			azureTenantId     = "tenant-id"
		)
		var (
			modelLibrary ModelLibrary
		)
		BeforeEach(func() {
			Expect(os.Setenv(envModelLibraryAzureClientSecret, azureClientSecret)).To(Succeed())
			Expect(os.Setenv(envModelLibraryAzureClientId, azureClientId)).To(Succeed())
			Expect(os.Setenv(envModelLibraryAzureTenantId, azureTenantId)).To(Succeed())
			modelLibrary = &azureModelLibrary{
				BaseModelLibrary: BaseModelLibrary{Uri: "uri"},
			}
		})

		When("Required env variable is not set", func() {
			It("Should return error", func() {
				Expect(os.Unsetenv(envModelLibraryAzureTenantId)).To(Succeed())
				Expect(modelLibrary.GetCredentials()).Error().To(HaveOccurred())
			})
		})

		When("All required env variables are set", func() {
			It("Should return a map containing the credentials", func() {
				credentials, err := modelLibrary.GetCredentials()
				Expect(err).ToNot(HaveOccurred())

				By("Containing as values the credentials provided as env variable")
				Expect(credentials).To(HaveKeyWithValue(envModelLibraryAzureClientSecret, azureClientSecret))
				Expect(credentials).To(HaveKeyWithValue(envModelLibraryAzureClientId, azureClientId))
				Expect(credentials).To(HaveKeyWithValue(envModelLibraryAzureTenantId, azureTenantId))
				By("Containing only the credentials provided as env variable")
				Expect(credentials).To(HaveLen(3))
			})
		})
	})
})
