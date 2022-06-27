package controllers

import (
	"fmt"
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("ModelDeployment controller", func() {
	const (
		timeout  = time.Second * 20
		interval = time.Second * 1
	)

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("When creating ModelDeployment", func() {
		const (
			modelDeploymentName               = "model-deployment-test"
			modelDeploymentNamespace          = "default"
			modelUri                          = "https://foo.bar/model.pkl"
			modelLibraryUri                   = "https://foo.bar/model-library"
			modelLibraryCredentialsSecretName = "azure-credentials"
			optimizationTarget                = n8sv1alpha1.OptimizationTargetLatency
			modelOptimizerImageVersion        = "0.0.1"
			modelOptimizerImageName           = "nebuly.ai/model-optimizer-mock"
		)

		It("Should start a model optimization job", func() {
			By("Creating a new ModelDeployment successfully")
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
					SourceModel: n8sv1alpha1.SourceModel{Uri: modelUri},
					ModelLibrary: n8sv1alpha1.ModelLibrary{
						Uri:        modelLibraryUri,
						Kind:       n8sv1alpha1.ModelLibraryKindAzure,
						SecretName: modelLibraryCredentialsSecretName,
					},
					Optimization: n8sv1alpha1.OptimizationSpec{
						Target:                     optimizationTarget,
						ModelOptimizerImageName:    modelOptimizerImageName,
						ModelOptimizerImageVersion: modelOptimizerImageVersion,
					},
				},
			}
			Expect(k8sClient.Create(ctx, modelDeployment)).To(Succeed())

			By("Checking the created ModelDeployment matches the specs")
			var createdModelDeployment n8sv1alpha1.ModelDeployment
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: modelDeploymentName, Namespace: modelDeploymentNamespace}
				if err := k8sClient.Get(ctx, lookupKey, &createdModelDeployment); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdModelDeployment.Spec).To(Equal(modelDeployment.Spec))

			By("Creating a new Job")
			Eventually(func() int {
				var jobList batchv1.JobList
				err := k8sClient.List(ctx, &jobList, client.MatchingLabels{LabelCreatedBy: modelDeploymentControllerName})
				if err != nil {
					return 0
				}
				return len(jobList.Items)
			}, timeout, interval).Should(Equal(1))

			By("Checking that the Job launched Pods using the specified Docker image")
			expectedImageName := fmt.Sprintf("%s:%s", modelOptimizerImageName, modelOptimizerImageVersion)
			var jobList batchv1.JobList
			Expect(k8sClient.List(ctx, &jobList, client.MatchingLabels{LabelCreatedBy: modelDeploymentControllerName})).To(Succeed())
			Expect(jobList.Items).To(HaveLen(1))
			job := jobList.Items[0]
			Expect(job.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedImageName))

			By("Checking that the Pods launched by the Job do not run as root")
			Expect(job.Spec.Template.Spec.SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
		})
	})
})
