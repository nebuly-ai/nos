package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ModelDeployment", func() {
	const (
		namespace           = "test"
		modelDeploymentName = "model-deployment-test"
		modelLibraryUri     = "https://foo.bar"
	)
	var _ = ModelDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: modelDeploymentName, Namespace: namespace},
		Spec:       ModelDeploymentSpec{},
	}

	//Describe("Get model library path", func() {
	//	It("Should contain the namespace", func() {
	//		Expect(m.GetModelLibraryPath()).To(ContainSubstring(namespace))
	//	})
	//	It("Should contain the name of the ModelDeployment", func() {
	//		Expect(m.GetModelLibraryPath()).To(ContainSubstring(modelDeploymentName))
	//	})
	//	It("Should contain the model library uri", func() {
	//		Expect(m.GetModelLibraryPath()).To(ContainSubstring(modelLibraryUri))
	//	})
	//})
})
