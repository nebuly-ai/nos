package controllers

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
		Expect(os.Setenv(EnvModelLibraryAzureClientSecret, azureClientSecret)).To(Succeed())
		Expect(os.Setenv(EnvModelLibraryAzureClientId, azureClientId)).To(Succeed())
		Expect(os.Setenv(EnvModelLibraryAzureTenantId, azureTenantId)).To(Succeed())
	})

	When("JSON config is empty", func() {
		It("Should return an error", func() {
			Expect(NewModelLibraryFromConfig("")).Error().To(HaveOccurred())
		})
	})

	When("JSON config is malformed", func() {
		It("Should return an error", func() {
			Expect(NewModelLibraryFromConfig("malformed")).Error().To(HaveOccurred())
		})
	})

	When("JSON has missing field", func() {
		It("Should return an error", func() {
			Expect(NewModelLibraryFromConfig("{\"foo\": \"bar\"")).Error().To(HaveOccurred())
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
			modelLibrary, err := NewModelLibraryFromConfig(json)
			Expect(err).ToNot(HaveOccurred())
			Expect(modelLibrary).ToNot(BeNil())
			var azureModelLibraryVar *azureModelLibrary
			Expect(modelLibrary).To(BeAssignableToTypeOf(azureModelLibraryVar))
		})
	})

	When("Model library kind is invalid", func() {
		json := newModelLibraryJson("https://foo.bar", "invalid")
		It("Should return an error", func() {
			Expect(NewModelLibraryFromConfig(json)).Error().To(HaveOccurred())
		})
	})

	When("Any required env variable is missing ", func() {
		json := newModelLibraryJson("https://foo.bar", string(ModelLibraryKindAzure))
		It("Should return an error", func() {
			Expect(os.Unsetenv(EnvModelLibraryAzureTenantId)).To(Succeed())
			Expect(NewModelLibraryFromConfig(json)).Error().To(HaveOccurred())
		})
	})

})
