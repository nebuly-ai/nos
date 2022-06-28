package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV1alpha1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V1alpha1 Suite")
}
