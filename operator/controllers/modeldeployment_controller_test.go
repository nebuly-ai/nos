package controllers

import (
	"fmt"
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/constants"
	"github.com/nebuly-ai/nebulnetes/controllers/reconcilers"
	"github.com/nebuly-ai/nebulnetes/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	modelDeploymentNamespace   = "default"
	modelUri                   = "https://foo.bar/model.pkl"
	optimizationTarget         = n8sv1alpha1.OptimizationTargetLatency
	modelOptimizerImageVersion = "0.0.1"
	modelOptimizerImageName    = "nebuly.ai/model-optimizer-mock"
)

func newMockedModelDeployment() *n8sv1alpha1.ModelDeployment {
	return &n8sv1alpha1.ModelDeployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: n8sv1alpha1.GroupVersion.String(),
			Kind:       "ModelDeployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.RandomStringLowercase(10),
			Namespace: modelDeploymentNamespace,
		},
		Spec: n8sv1alpha1.ModelDeploymentSpec{
			SourceModel: n8sv1alpha1.SourceModel{Uri: modelUri},
			Optimization: n8sv1alpha1.OptimizationSpec{
				Target:                     optimizationTarget,
				ModelOptimizerImageName:    modelOptimizerImageName,
				ModelOptimizerImageVersion: modelOptimizerImageVersion,
			},
		},
	}
}

func getOptimizationJobList(modelDeployment *n8sv1alpha1.ModelDeployment) (*batchv1.JobList, error) {
	var jobList = new(batchv1.JobList)
	labels := client.MatchingLabels{
		constants.LabelCreatedBy:       constants.ModelDeploymentControllerName,
		constants.LabelModelDeployment: modelDeployment.GetName(),
	}
	err := k8sClient.List(ctx, jobList, labels)
	if err != nil {
		return nil, err
	}
	return jobList, nil
}

var _ = Describe("ModelDeployment controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
		if utils.GetEnvBool(constants.EnvSkipControllerTests, false) {
			Skip(fmt.Sprintf("%s is true, skipping controller tests", constants.EnvSkipControllerTests))
		}
	})

	AfterEach(func() {
		// Add any setup steps that needs to be executed after each test
	})

	Context("When creating ModelDeployment", func() {
		It("Should start a model optimization job", func() {
			By("Creating a new ModelDeployment successfully")
			modelDeployment := newMockedModelDeployment()
			Expect(k8sClient.Create(ctx, modelDeployment)).To(Succeed())

			By("Checking the created ModelDeployment matches the specs")
			var createdModelDeployment = new(n8sv1alpha1.ModelDeployment)
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: modelDeployment.Name, Namespace: modelDeployment.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, createdModelDeployment); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdModelDeployment.Spec).To(Equal(modelDeployment.Spec))

			By("Creating a new Job")
			Eventually(func() int {
				jobList, err := getOptimizationJobList(modelDeployment)
				if err != nil {
					return 0
				}
				return len(jobList.Items)
			}, timeout, interval).Should(Equal(1))

			By("Checking that the Job launched Pods using the specified Docker image")
			expectedImageName := fmt.Sprintf("%s:%s", modelOptimizerImageName, modelOptimizerImageVersion)
			jobList, err := getOptimizationJobList(modelDeployment)
			Expect(err).ToNot(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(1))
			job := jobList.Items[0]
			Expect(job.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedImageName))

			By("Checking that the Pods launched by the Job do not run as root")
			Expect(job.Spec.Template.Spec.SecurityContext.RunAsNonRoot).To(HaveValue(BeTrue()))
		})
	})

	Context("When updating the optimization target of a ModelDeployment", func() {
		It("Should delete and recreate the model optimization job", func() {
			By("Creating a new ModelDeployment successfully")
			modelDeployment := newMockedModelDeployment()
			Expect(k8sClient.Create(ctx, modelDeployment)).To(Succeed())

			By("Getting the optimization Job created with the first optimization target")
			Eventually(func() int {
				jobList, err := getOptimizationJobList(modelDeployment)
				if err != nil {
					return 0
				}
				return len(jobList.Items)
			}, timeout, interval).Should(Equal(1))

			By("Checking the created ModelDeployment matches the specs")
			var createdModelDeployment = new(n8sv1alpha1.ModelDeployment)
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: modelDeployment.Name, Namespace: modelDeployment.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, createdModelDeployment); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdModelDeployment.Spec).To(Equal(modelDeployment.Spec))

			By("Updating the optimization target of the ModelDeployment")
			createdModelDeployment.Spec.Optimization.Target = n8sv1alpha1.OptimizationTargetEmissions
			Expect(k8sClient.Update(ctx, createdModelDeployment)).To(Succeed())

			By("Checking that the old job gets marked for deletion (e.g. deletion timestamp != 0)")
			Eventually(func() bool {
				jobList, err := getOptimizationJobList(modelDeployment)
				if err != nil {
					return false
				}
				if len(jobList.Items) == 1 {
					return !jobList.Items[0].GetDeletionTimestamp().IsZero()
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When updating the source model URI of a ModelDeployment", func() {
		It("Should delete and recreate the model optimization job", func() {
			By("Creating a new ModelDeployment successfully")
			modelDeployment := newMockedModelDeployment()
			Expect(k8sClient.Create(ctx, modelDeployment)).To(Succeed())

			By("Getting the optimization Job created with the first optimization target")
			Eventually(func() int {
				jobList, err := getOptimizationJobList(modelDeployment)
				if err != nil {
					return 0
				}
				return len(jobList.Items)
			}, timeout, interval).Should(Equal(1))

			By("Checking the created ModelDeployment matches the specs")
			var createdModelDeployment = new(n8sv1alpha1.ModelDeployment)
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: modelDeployment.Name, Namespace: modelDeployment.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, createdModelDeployment); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdModelDeployment.Spec).To(Equal(modelDeployment.Spec))
			By("Updating the optimization target of the ModelDeployment")

			createdModelDeployment.Spec.Optimization.Target = n8sv1alpha1.OptimizationTargetEmissions
			Expect(k8sClient.Update(ctx, createdModelDeployment)).To(Succeed())

			By("Checking that the old job gets marked for deletion (e.g. deletion timestamp != 0)")
			Eventually(func() bool {
				jobList, err := getOptimizationJobList(modelDeployment)
				if err != nil {
					return false
				}
				if len(jobList.Items) == 1 {
					return !jobList.Items[0].GetDeletionTimestamp().IsZero()
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When the optimization Job of a ModelDeployment completes successfully", func() {
		It("Should create a ConfigMap containing the model descriptor of the optimized model", func() {
			By("Creating a new ModelDeployment successfully")
			modelDeployment := newMockedModelDeployment()
			Expect(k8sClient.Create(ctx, modelDeployment)).To(Succeed())

			By("Getting the optimization Job created with the first optimization target")
			var optimizationJob = new(batchv1.Job)
			Eventually(func() int {
				jobList, err := getOptimizationJobList(modelDeployment)
				if err != nil {
					return 0
				}
				length := len(jobList.Items)
				if length == 1 {
					optimizationJob = &jobList.Items[0]
				}
				return length
			}, timeout, interval).Should(Equal(1))
			Expect(optimizationJob.Name).ToNot(BeEmpty())

			By("Updating the Job status to completed")
			optimizationJob.Status.Conditions = append(optimizationJob.Status.Conditions, batchv1.JobCondition{
				Type:               batchv1.JobComplete,
				Status:             corev1.ConditionTrue,
				LastProbeTime:      metav1.Time{},
				LastTransitionTime: metav1.Time{},
				Reason:             "",
				Message:            "",
			})
			Expect(k8sClient.Status().Update(ctx, optimizationJob)).To(Succeed())

			Eventually(func() int {
				jobList, err := getOptimizationJobList(modelDeployment)
				if err != nil {
					return 0
				}
				length := len(jobList.Items)
				if length == 1 {
					optimizationJob = &jobList.Items[0]
				}
				return length
			}, timeout, interval).Should(Equal(1))

			By("Creating a ConfigMap containing the optimized model descriptor")
			var configMap corev1.ConfigMap
			Eventually(func() bool {
				var configMapList = new(corev1.ConfigMapList)
				labels := client.MatchingLabels{}
				for k, v := range reconcilers.GetModelDescriptorConfigMapLabels(modelDeployment, optimizationJob) {
					labels[k] = v
				}
				err := k8sClient.List(ctx, configMapList, labels)
				if err != nil {
					return false
				}
				if len(configMapList.Items) == 1 {
					configMap = configMapList.Items[0]
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
			Expect(configMap.Name).ToNot(BeEmpty())
			Expect(configMap.Data).To(HaveKeyWithValue("model_uri", modelUri))
		})
	})
})
