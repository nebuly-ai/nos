package controller

import (
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"time"
)

var _ = Describe("ElasticQuota controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1

		elasticQuotaNamespace   = "default"
		elasticQuotaName        = "test-elasticquota"
		elasticQuotaMinCPUMilli = 4000
		elasticQuotaMaxCPUMilli = 6000

		containerOneCPUMilli = 500
		containerTwoCPUMilli = 500
	)

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
		//if utils.GetEnvBool(constants.EnvSkipControllerTests, false) {
		//	Skip(fmt.Sprintf("%s is true, skipping controller tests", constants.EnvSkipControllerTests))
		//}
	})

	AfterEach(func() {
		// Add any setup steps that needs to be executed after each test
	})

	When("New pods belonging to ElasticQuota namespace are in running status", func() {
		It("Should update the ElasticQuota status", func() {
			By("Creating an ElasticQuota successfully")
			elasticQuota := v1alpha1.ElasticQuota{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ElasticQuota",
					APIVersion: v1alpha1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      elasticQuotaName,
					Namespace: elasticQuotaNamespace,
				},
				Spec: v1alpha1.ElasticQuotaSpec{
					Min: util.ResourceList(&framework.Resource{
						MilliCPU: elasticQuotaMinCPUMilli,
					}),
					Max: util.ResourceList(&framework.Resource{
						MilliCPU: elasticQuotaMaxCPUMilli,
					}),
				},
			}
			Expect(k8sClient.Create(ctx, &elasticQuota)).To(Succeed())

			By("Checking the created ElasticQuota matches the specs")
			var instance = v1alpha1.ElasticQuota{}
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, &instance); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(instance.Spec).To(Equal(elasticQuota.Spec))

			By("Creating a new Pod within the ElasticQuota namespace")
			pod := v1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: v1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      util.RandomStringLowercase(5),
					Namespace: elasticQuotaNamespace,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "container-1",
							Image: "test:0.0.1",
							Resources: v1.ResourceRequirements{
								Limits: util.ResourceListForPod(&framework.Resource{
									MilliCPU: containerOneCPUMilli,
								}),
							},
						},
						{
							Name:  "container-2",
							Image: "test:0.0.1",
							Resources: v1.ResourceRequirements{
								Limits: util.ResourceListForPod(&framework.Resource{
									MilliCPU: containerTwoCPUMilli,
								}),
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &pod)).To(Succeed())

			By("Not updating the ElasticQuota status until the Pod is in running state")
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, &instance); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(instance.Status.Used.Cpu().String()).To(Equal("0"))

			By("Updating the Pod status to running")
			pod.Status.Phase = v1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			By("Checking that the ElasticQuota status gets updated")
			expectedUsedResourceList := util.ResourceListForPod(&framework.Resource{
				MilliCPU: containerTwoCPUMilli + containerTwoCPUMilli,
			})
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, &instance); err != nil {
					return false
				}
				return instance.Status.Used.Cpu().String() == expectedUsedResourceList.Cpu().String()
			}, timeout, interval).Should(BeTrue())
		})
	})
})
