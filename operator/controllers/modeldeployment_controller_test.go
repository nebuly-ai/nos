package controllers

import (
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("ModelDeployment controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Second * 1
	)

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("When creating ModelDeployment", func() {
		It("Should launch a new model optimization job", func() {
			var (
				modelDeploymentName      = "model-deployment-test"
				modelDeploymentNamespace = "default"
				modelUri                 = "https://foo.bar/model.pkl"
				modelLibraryUri          = "https://foo.bar/model-library"
				optimizationTarget       = n8sv1alpha1.OptimizationTargetLatency
				modelOptimizerVersion    = "9.9.9"
			)

			By("Creating a new ModelDeployment")
			modelDeployment := &n8sv1alpha1.ModelDeployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: n8sv1alpha1.GroupVersion.String(),
					Kind:       "ModelDeployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      modelDeploymentName,
					Namespace: modelDeploymentNamespace,
				},
				Spec: n8sv1alpha1.ModelDeploymentSpec{
					ModelUri:        modelUri,
					ModelLibraryUri: modelLibraryUri,
					Optimization: n8sv1alpha1.OptimizationSpec{
						Target:                optimizationTarget,
						ModelOptimizerVersion: modelOptimizerVersion,
					},
				},
			}
			Expect(k8sClient.Create(ctx, modelDeployment)).Should(Succeed())

			By("Checking the created ModelDeployment matches the specs")
			// fetch the newly created model deployment
			var createdModelDeployment n8sv1alpha1.ModelDeployment
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: modelDeploymentName, Namespace: modelDeploymentNamespace}
				if err := k8sClient.Get(ctx, lookupKey, &createdModelDeployment); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			// check if the actual object matches the specs
			Expect(createdModelDeployment.Spec).Should(Equal(modelDeployment.Spec))
		})
	})
})
