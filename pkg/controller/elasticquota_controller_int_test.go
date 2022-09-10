//go:build integration

package controller

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
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
		namespace = factory.BuildNamespace(util.RandomStringLowercase(16)).Get()
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
			Eventually(func() v1alpha1.ElasticQuotaSpec {
				lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
				_ = k8sClient.Get(ctx, lookupKey, &instance)
				return instance.Spec
			}, timeout, interval).Should(Equal(elasticQuota.Spec))

			By("Creating a new Pod within the ElasticQuota namespace")
			pod := factory.BuildPod(elasticQuota.Namespace, "pod-1").
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
			Eventually(func() string {
				lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
				if err := k8sClient.Get(ctx, lookupKey, &instance); err != nil {
					return ""
				}
				return instance.Status.Used.Cpu().String()
			}, timeout, interval).Should(Equal("0"))

			By("Creating a new Namespace")
			newNamespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			Expect(k8sClient.Create(ctx, &newNamespace)).To(Succeed())

			By("Creating another Pod belonging to the new namespace")
			otherNamespacePod := factory.BuildPod(newNamespace.Name, "pod-2").
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
			Eventually(func() v1alpha1.ElasticQuota {
				lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
				_ = k8sClient.Get(ctx, lookupKey, &instance)
				return instance
			}, timeout, interval).ShouldNot(Equal(previousInstance))
			Expect(instance.Status.Used).To(Equal(expectedUsedResourceList))

			By("Checking the Pod's capacity-info label shows that the Pod is in in-quota")
			assertPodHasLabel(
				ctx,
				pod,
				constant.LabelCapacityInfo,
				string(constant.CapacityInfoInQuota),
				timeout,
				interval,
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
			anotherNamespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			anotherElasticQuota := factory.BuildEq(anotherNamespace.Name, util.RandomStringLowercase(10)).
				WithMinGPUMemory(100).
				WithMaxGPUMemory(100).
				Get()
			Expect(k8sClient.Create(ctx, &anotherNamespace)).To(Succeed())
			Expect(k8sClient.Create(ctx, &anotherElasticQuota)).To(Succeed())

			By("Creating a new Pod within the ElasticQuota namespace, requesting GPUMemory > EQ min")
			pod := factory.BuildPod(elasticQuota.Namespace, "pod-2").
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
			assertPodHasLabel(
				ctx,
				pod,
				constant.LabelCapacityInfo,
				string(constant.CapacityInfoOverQuota),
				timeout,
				interval,
			)
		})

		When("A Pod is in over-quota and some other Pod stops running freeing up available quotas", func() {
			It("Should update the Pod capacity info label from over-quota to in-quota", func() {
				const (
					elasticQuotaMinGPUMemory = 4
					elasticQuotaMaxGPUMemory = 6
					inQuotaPodGPUMemory      = elasticQuotaMinGPUMemory - 2 // since pods are created at the same time, in-quota Pod GPU request must be < over-quota Pod GPU request so that they get labelled properly
					overQuotaPodGPUMemory    = elasticQuotaMinGPUMemory - 1
					elasticQuotaName         = "test-elasticquota"
				)

				By("Creating an ElasticQuota successfully")
				elasticQuota := factory.BuildEq(namespace.Name, elasticQuotaName).
					WithMinGPUMemory(elasticQuotaMinGPUMemory).
					WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
					Get()
				Expect(k8sClient.Create(ctx, &elasticQuota)).To(Succeed())

				By("Creating another ElasticQuota with high GPUMemory min successfully")
				anotherNamespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
				anotherElasticQuota := factory.BuildEq(anotherNamespace.Name, util.RandomStringLowercase(10)).
					WithMinGPUMemory(100).
					WithMaxGPUMemory(100).
					Get()
				Expect(k8sClient.Create(ctx, &anotherNamespace)).To(Succeed())
				Expect(k8sClient.Create(ctx, &anotherElasticQuota)).To(Succeed())

				By("Creating a new Pod within the ElasticQuota namespace, requesting GPUMemory < EQ min")
				inQuotaPod := factory.BuildPod(elasticQuota.Namespace, "pod-1").
					WithContainer(
						factory.BuildContainer("container-1", "test:0.0.1").
							WithGPUMemoryLimit(inQuotaPodGPUMemory).
							Get(),
					).
					Get()
				Expect(k8sClient.Create(ctx, &inQuotaPod)).To(Succeed())

				By("Updating the in-quota Pod status to running")
				inQuotaPod.Status.Phase = v1.PodRunning
				Expect(k8sClient.Status().Update(ctx, &inQuotaPod)).To(Succeed())

				By("Creating a new over-quota Pod within the ElasticQuota namespace, requesting GPUMemory < EQ min")
				overQuotaPod := factory.BuildPod(elasticQuota.Namespace, "pod-2").
					WithContainer(
						factory.BuildContainer("container-1", "test:0.0.1").
							WithGPUMemoryLimit(overQuotaPodGPUMemory).
							Get(),
					).
					Get()
				Expect(k8sClient.Create(ctx, &overQuotaPod)).To(Succeed())

				By("Updating the over-quota Pod status to running")
				overQuotaPod.Status.Phase = v1.PodRunning
				Expect(k8sClient.Status().Update(ctx, &overQuotaPod)).To(Succeed())

				By("Checking the ElasticQuota status")
				var eqInstance v1alpha1.ElasticQuota
				Eventually(func() string {
					lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
					if err := k8sClient.Get(ctx, lookupKey, &eqInstance); err != nil {
						logger.Error(err, "unable to fetch ElasticQuota", "lookup-key", lookupKey)
						return ""
					}
					return eqInstance.Status.Used.Name(v1alpha1.ResourceGPUMemory, resource.DecimalSI).String()
				}, timeout, interval).Should(Equal(strconv.Itoa(inQuotaPodGPUMemory + overQuotaPodGPUMemory)))

				By("Checking the in-quota Pod's capacity-info label is in-quota")
				assertPodHasLabel(
					ctx,
					inQuotaPod,
					constant.LabelCapacityInfo,
					string(constant.CapacityInfoInQuota),
					timeout,
					interval,
				)

				By("Checking the over-quota Pod's capacity-info label is over-quota")
				assertPodHasLabel(
					ctx,
					overQuotaPod,
					constant.LabelCapacityInfo,
					string(constant.CapacityInfoOverQuota),
					timeout,
					interval,
				)

				By("Updating the in-quota Pod status to succeeded")
				original := inQuotaPod.DeepCopy()
				inQuotaPod.Status.Phase = v1.PodSucceeded
				Expect(k8sClient.Status().Patch(ctx, &inQuotaPod, client.MergeFrom(original))).To(Succeed())

				By("Checking the over-quota is now in-quota")
				assertPodHasLabel(
					ctx,
					overQuotaPod,
					constant.LabelCapacityInfo,
					string(constant.CapacityInfoInQuota),
					timeout,
					interval,
				)
			})

			When("An ElasticQuota min field is updated", func() {
				It("Should update Pods capacity-info label accordingly", func() {
					const (
						elasticQuotaMinGPUMemory            = 4
						elasticQuotaMinGPUMemoryAfterUpdate = 2
						elasticQuotaMaxGPUMemory            = 6

						podOneGPUMemory = 2
						podTwoGPUMemory = 2
					)

					By("Creating an ElasticQuota successfully")
					elasticQuota := factory.BuildEq(namespace.Name, util.RandomStringLowercase(10)).
						WithMinGPUMemory(elasticQuotaMinGPUMemory).
						WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
						Get()
					Expect(k8sClient.Create(ctx, &elasticQuota)).To(Succeed())

					By("Creating new Pods within the ElasticQuota namespace, requesting total GPUMemory <= EQ min")
					podOne := factory.BuildPod(elasticQuota.Namespace, "pod-1").
						WithContainer(
							factory.BuildContainer("container-1", "test:0.0.1").
								WithGPUMemoryLimit(podOneGPUMemory).
								Get(),
						).
						Get()
					podTwo := factory.BuildPod(elasticQuota.Namespace, "pod-2").
						WithContainer(
							factory.BuildContainer("container-1", "test:0.0.1").
								WithGPUMemoryLimit(podTwoGPUMemory).
								Get(),
						).
						Get()
					Expect(k8sClient.Create(ctx, &podOne)).To(Succeed())
					Expect(k8sClient.Create(ctx, &podTwo)).To(Succeed())

					By("Updating the status phase of the Pods to running")
					podOne.Status.Phase = v1.PodRunning
					podTwo.Status.Phase = v1.PodRunning
					Expect(k8sClient.Status().Update(ctx, &podOne)).To(Succeed())
					Expect(k8sClient.Status().Update(ctx, &podTwo)).To(Succeed())

					By("Checking that both the Pods are labelled as in-quota")
					assertPodHasLabel(
						ctx,
						podOne,
						constant.LabelCapacityInfo,
						string(constant.CapacityInfoInQuota),
						timeout,
						interval,
					)
					assertPodHasLabel(
						ctx,
						podTwo,
						constant.LabelCapacityInfo,
						string(constant.CapacityInfoInQuota),
						timeout,
						interval,
					)

					By("Updating the ElasticQuota reducing the min value")
					original := elasticQuota.DeepCopy()
					elasticQuota.Spec.Min[v1alpha1.ResourceGPUMemory] = *resource.NewQuantity(elasticQuotaMinGPUMemoryAfterUpdate, resource.DecimalSI)
					Expect(k8sClient.Patch(ctx, &elasticQuota, client.MergeFrom(original))).To(Succeed())

					By("Checking that the Pod created first is still labelled as in-quota")
					assertPodHasLabel(
						ctx,
						podOne,
						constant.LabelCapacityInfo,
						string(constant.CapacityInfoInQuota),
						timeout,
						interval,
					)

					By("Checking that the Pod created last is now labelled as over-quota")
					assertPodHasLabel(
						ctx,
						podTwo,
						constant.LabelCapacityInfo,
						string(constant.CapacityInfoOverQuota),
						timeout,
						interval,
					)
				})
			})
		})
	})
})

func assertPodHasLabel(ctx context.Context, pod v1.Pod, label, value string, timeout, interval time.Duration) {
	var instance v1.Pod
	Eventually(func(g Gomega) {
		lookupKey := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
		g.Expect(k8sClient.Get(ctx, lookupKey, &instance)).To(Succeed())
		g.Expect(instance.Labels).To(HaveKeyWithValue(label, value))
	}, timeout, interval).Should(Succeed())
}
