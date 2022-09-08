//go:build integration

package controller

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/factory"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"time"
)

var _ = Describe("ElasticQuota controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	// Dedicated namespace for each test
	var namespace v1.Namespace

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
		//if utils.GetEnvBool(constants.EnvSkipControllerTests, false) {
		//	Skip(fmt.Sprintf("%s is true, skipping controller tests", constants.EnvSkipControllerTests))
		//}
		namespace = factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
		Expect(k8sClient.Create(ctx, &namespace)).To(Succeed())
	})

	AfterEach(func() {
		// Add any setup steps that needs to be executed after each test
		Expect(k8sClient.Delete(ctx, &namespace)).To(Succeed())
	})

	When("New pods belonging to ElasticQuota namespace are in running status", func() {
		It("Should update the ElasticQuota status", func() {
			const (
				elasticQuotaMinCPUMilli  = 4000
				elasticQuotaMinGPUMemory = 4
				elasticQuotaMaxCPUMilli  = 6000
				elasticQuotaMaxGPUMemory = 6
				elasticQuotaName         = "test-elasticquota"

				containerOneCPUMilli  = 500
				containerOneGPUMemory = 1
				containerTwoCPUMilli  = 500
				containerTwoGPUMemory = 2
			)
			By("Creating an ElasticQuota successfully")
			elasticQuota := factory.BuildEq(namespace.Name, elasticQuotaName).
				WithMinCPUMilli(elasticQuotaMinCPUMilli).
				WithMinGPUMemory(elasticQuotaMinGPUMemory).
				WithMaxCPUMilli(elasticQuotaMaxCPUMilli).
				WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
				Get()
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
			pod := factory.BuildPod(elasticQuota.Namespace, util.RandomStringLowercase(5)).
				WithContainer(
					factory.BuildContainer("container-1", "test:0.0.1").
						WithCPUMilliLimit(containerOneCPUMilli).
						WithGPUMemoryLimit(containerOneGPUMemory).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("container-2", "test:0.0.1").
						WithCPUMilliLimit(containerTwoCPUMilli).
						WithGPUMemoryLimit(containerTwoGPUMemory).
						Get(),
				).
				Get()
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

			By("Creating a new Namespace")
			newNamespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			Expect(k8sClient.Create(ctx, &newNamespace)).To(Succeed())

			By("Creating another Pod belonging to the new namespace")
			otherNamespacePod := factory.BuildPod(newNamespace.Name, util.RandomStringLowercase(5)).
				WithContainer(
					factory.BuildContainer("container-1", "test:0.0.1").
						WithCPUMilliLimit(containerOneCPUMilli).
						WithGPUMemoryLimit(containerOneGPUMemory).
						Get(),
				).Get()
			Expect(k8sClient.Create(ctx, &otherNamespacePod)).To(Succeed())

			By("Updating the Pods status to running")
			pod.Status.Phase = v1.PodRunning
			otherNamespacePod.Status.Phase = v1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())
			Expect(k8sClient.Status().Update(ctx, &otherNamespacePod)).To(Succeed())

			By("Checking that the ElasticQuota status gets updated only considering the Pods in its namespace")
			expectedCPUQuantity, _ := resource.ParseQuantity(fmt.Sprintf("%d", (containerOneCPUMilli+containerTwoCPUMilli)/1000))
			expectedGPUMemoryQuantity, _ := resource.ParseQuantity(fmt.Sprintf("%d", containerOneGPUMemory+containerTwoGPUMemory))
			expectedUsedResourceList := v1.ResourceList{
				v1.ResourceCPU:             expectedCPUQuantity,
				v1alpha1.ResourceGPUMemory: expectedGPUMemoryQuantity,
			}
			previousInstance := instance
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, &instance); err != nil {
					return false
				}
				updated := !reflect.DeepEqual(previousInstance, instance)
				return updated
			}, timeout, interval).Should(BeTrue())
			Expect(instance.Status.Used).To(Equal(expectedUsedResourceList))

			By("Checking the Pod's capacity-info label shows that the Pod is in in-quota")
			var podInstance v1.Pod
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: pod.Name, Namespace: elasticQuota.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, &podInstance); err != nil {
					return false
				}
				if capacityInfo, ok := podInstance.Labels[constant.LabelCapacityInfo]; ok {
					return capacityInfo == string(constant.CapacityInfoInQuota)
				}
				return false
			}, timeout, interval).Should(
				BeTrue(),
				fmt.Sprintf("Label %q should have value %q", constant.LabelCapacityInfo, constant.CapacityInfoInQuota),
			)

			By("Checking that the Pods in other namespaces do not get labelled with EQ capacity info")
			var otherNamespacePodInstance v1.Pod
			lookupKey := types.NamespacedName{Name: otherNamespacePod.Name, Namespace: otherNamespacePod.Namespace}
			Expect(k8sClient.Get(ctx, lookupKey, &otherNamespacePodInstance)).To(Succeed())
			Expect(otherNamespacePodInstance.Labels).ToNot(ContainElement(constant.LabelCapacityInfo))
		})
	})

	When("A new Pod is scheduled in over-quota", func() {
		It("Should add a label specifying that the Pod is using over-quotas and can thus be preempted", func() {
			const (
				elasticQuotaMinGPUMemory = 4
				elasticQuotaMaxGPUMemory = 6
				elasticQuotaName         = "test-elasticquota"
			)

			By("Creating an ElasticQuota successfully")
			elasticQuota := factory.BuildEq(namespace.Name, elasticQuotaName).
				WithMinGPUMemory(elasticQuotaMinGPUMemory).
				WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
				Get()
			Expect(k8sClient.Create(ctx, &elasticQuota)).To(Succeed())

			By("Creating another ElasticQuota with high GPUMemory min successfully")
			anotherElasticQuota := factory.BuildEq(namespace.Name, util.RandomStringLowercase(10)).
				WithMinGPUMemory(100).
				WithMaxGPUMemory(100).
				Get()
			Expect(k8sClient.Create(ctx, &anotherElasticQuota)).To(Succeed())

			By("Creating a new Pod within the ElasticQuota namespace, requesting GPUMemory > EQ min")
			pod := factory.BuildPod(elasticQuota.Namespace, util.RandomStringLowercase(5)).
				WithContainer(
					factory.BuildContainer("container-1", "test:0.0.1").
						WithGPUMemoryLimit(5).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("container-2", "test:0.0.1").
						WithGPUMemoryLimit(5).
						Get(),
				).
				Get()
			Expect(k8sClient.Create(ctx, &pod)).To(Succeed())

			By("Updating the Pod status to running")
			pod.Status.Phase = v1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())

			By("Checking the Pod's capacity-info label shows that the Pod is in over-quota")
			var podInstance v1.Pod
			Eventually(func() bool {
				lookupKey := types.NamespacedName{Name: pod.Name, Namespace: elasticQuota.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, &podInstance); err != nil {
					return false
				}
				if capacityInfo, ok := podInstance.Labels[constant.LabelCapacityInfo]; ok {
					return capacityInfo == string(constant.CapacityInfoOverQuota)
				}
				return false
			}, timeout, interval).Should(
				BeTrue(),
				fmt.Sprintf("Label %q should have value %q", constant.LabelCapacityInfo, constant.CapacityInfoOverQuota),
			)
		})
	})
})
