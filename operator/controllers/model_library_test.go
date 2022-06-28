package controllers

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("New model library from json config", func() {
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
		It("Should ignore extra fields and return a ModelLibrary struct with the values specified in the JSON", func() {
			modelLibrary, err := NewModelLibraryFromConfig(json)
			Expect(err).ToNot(HaveOccurred())
			Expect(modelLibrary.Uri).To(Equal(uri))
			Expect(modelLibrary.Kind).To(Equal(kind))
		})
	})
})
