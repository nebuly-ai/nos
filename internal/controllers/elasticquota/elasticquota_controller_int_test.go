//go:build integration

/*
 * Copyright 2023 nebuly.com.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package elasticquota

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.com/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/test/factory"
	"github.com/nebuly-ai/nos/pkg/util"
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
		namespace = factory.BuildNamespace(util.RandomStringLowercase(16)).Get()
		Expect(k8sClient.Create(ctx, &namespace)).To(Succeed())
	})

	AfterEach(func() {
		// Add any setup steps that needs to be executed after each test
	})

	When("New pods belonging to ElasticQuota namespace are in running status", func() {
		It("Should update the ElasticQuota status", func() {
			const (
				elasticQuotaMinCPUMilli  = 4000
				elasticQuotaMinGPUMemory = 4 * constant.DefaultNvidiaGPUResourceMemory
				elasticQuotaMaxCPUMilli  = 6000
				elasticQuotaMaxGPUMemory = 5 * constant.DefaultNvidiaGPUResourceMemory
				elasticQuotaName         = "test-elasticquota"

				containerOneCPUMilli        = 500
				containerOneNvidiaGpu       = 1
				containerTwoCPUMilli        = 500
				containerTwoNvidiaGpu       = 2
				containerTwoNvidiaMigMemory = 1
			)
			var containerTwoNvidiaMigResource = v1.ResourceName(
				fmt.Sprintf("nvidia.com/mig-1g.%dgb", containerTwoNvidiaMigMemory),
			)

			By("Creating an ElasticQuota successfully")
			elasticQuota := v1alpha1.BuildEq(namespace.Name, elasticQuotaName).
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
						WithNvidiaGPULimit(containerOneNvidiaGpu).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("container-2", "test:0.0.1").
						WithCPUMilliLimit(containerTwoCPUMilli).
						WithNvidiaGPULimit(containerTwoNvidiaGpu).
						WithScalarResourceLimit(containerTwoNvidiaMigResource, 1).
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
						WithNvidiaGPULimit(containerOneNvidiaGpu).
						Get(),
				).Get()
			Expect(k8sClient.Create(ctx, &otherNamespacePod)).To(Succeed())

			By("Updating the Pods status to running")
			pod.Status.Phase = v1.PodRunning
			otherNamespacePod.Status.Phase = v1.PodRunning
			Expect(k8sClient.Status().Update(ctx, &pod)).To(Succeed())
			Expect(k8sClient.Status().Update(ctx, &otherNamespacePod)).To(Succeed())

			By("Checking that the ElasticQuota status gets updated only considering (1) the Pods in its namespace and (2) the resources defined in the quota spec")
			expectedCPUQuantity, _ := resource.ParseQuantity(
				strconv.Itoa((containerOneCPUMilli + containerTwoCPUMilli) / 1000),
			)
			expectedGPUMemoryQuantity, _ := resource.ParseQuantity(
				strconv.Itoa((containerOneNvidiaGpu+containerTwoNvidiaGpu)*constant.DefaultNvidiaGPUResourceMemory + containerTwoNvidiaMigMemory),
			)
			expectedUsedResourceList := v1.ResourceList{
				v1.ResourceCPU:             expectedCPUQuantity,
				v1alpha1.ResourceGPUMemory: expectedGPUMemoryQuantity,
			}
			Eventually(func() v1.ResourceList {
				lookupKey := types.NamespacedName{Name: elasticQuota.Name, Namespace: elasticQuota.Namespace}
				_ = k8sClient.Get(ctx, lookupKey, &instance)
				return instance.Status.Used
			}, timeout, interval).Should(Equal(expectedUsedResourceList))

			By("Checking the Pod's capacity-info label shows that the Pod is in in-quota")
			assertPodHasLabel(
				ctx,
				pod,
				v1alpha1.LabelCapacityInfo,
				string(constant.CapacityInfoInQuota),
				timeout,
				interval,
			)

			By("Checking that the Pods in other namespaces do not get labelled with EQ capacity info")
			var otherNamespacePodInstance v1.Pod
			lookupKey := types.NamespacedName{Name: otherNamespacePod.Name, Namespace: otherNamespacePod.Namespace}
			Expect(k8sClient.Get(ctx, lookupKey, &otherNamespacePodInstance)).To(Succeed())
			Expect(otherNamespacePodInstance.Labels).ToNot(ContainElement(v1alpha1.LabelCapacityInfo))
		})
	})

	When("A new Pod is scheduled in over-quota", func() {
		It("Should add a label specifying that the Pod is using over-quotas and can thus be preempted", func() {
			const (
				elasticQuotaMinGPUMemory = 4 * constant.DefaultNvidiaGPUResourceMemory
				elasticQuotaMaxGPUMemory = 6 * constant.DefaultNvidiaGPUResourceMemory
				elasticQuotaName         = "test-elasticquota"
			)

			By("Creating an ElasticQuota successfully")
			elasticQuota := v1alpha1.BuildEq(namespace.Name, elasticQuotaName).
				WithMinGPUMemory(elasticQuotaMinGPUMemory).
				WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
				Get()
			Expect(k8sClient.Create(ctx, &elasticQuota)).To(Succeed())

			By("Creating another ElasticQuota with high GPUMemory min successfully")
			anotherNamespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
			anotherElasticQuota := v1alpha1.BuildEq(anotherNamespace.Name, util.RandomStringLowercase(10)).
				WithMinGPUMemory(100 * constant.DefaultNvidiaGPUResourceMemory).
				WithMaxGPUMemory(100 * constant.DefaultNvidiaGPUResourceMemory).
				Get()
			Expect(k8sClient.Create(ctx, &anotherNamespace)).To(Succeed())
			Expect(k8sClient.Create(ctx, &anotherElasticQuota)).To(Succeed())

			By("Creating a new Pod within the ElasticQuota namespace, requesting GPUMemory > EQ min")
			pod := factory.BuildPod(elasticQuota.Namespace, "pod-2").
				WithContainer(
					factory.BuildContainer("container-1", "test:0.0.1").
						WithNvidiaGPULimit(5).
						Get(),
				).
				WithContainer(
					factory.BuildContainer("container-2", "test:0.0.1").
						WithNvidiaGPULimit(5).
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
				v1alpha1.LabelCapacityInfo,
				string(constant.CapacityInfoOverQuota),
				timeout,
				interval,
			)
		})

		When("A Pod is in over-quota and some other Pod stops running freeing up available quotas", func() {
			It("Should update the Pod capacity info label from over-quota to in-quota", func() {
				const (
					elasticQuotaMinGPUMemory = 4 * constant.DefaultNvidiaGPUResourceMemory
					elasticQuotaMaxGPUMemory = 6 * constant.DefaultNvidiaGPUResourceMemory
					inQuotaPodGPU            = 4 - 2 // since pods are created at the same time, in-quota Pod GPU request must be < over-quota Pod GPU request so that they get labelled properly
					overQuotaPodGPU          = 4 - 1
					elasticQuotaName         = "test-elasticquota"
				)

				By("Creating an ElasticQuota successfully")
				elasticQuota := v1alpha1.BuildEq(namespace.Name, elasticQuotaName).
					WithMinGPUMemory(elasticQuotaMinGPUMemory).
					WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
					Get()
				Expect(k8sClient.Create(ctx, &elasticQuota)).To(Succeed())

				By("Creating another ElasticQuota with high GPUMemory min successfully")
				anotherNamespace := factory.BuildNamespace(util.RandomStringLowercase(10)).Get()
				anotherElasticQuota := v1alpha1.BuildEq(anotherNamespace.Name, util.RandomStringLowercase(10)).
					WithMinGPUMemory(100 * constant.DefaultNvidiaGPUResourceMemory).
					WithMaxGPUMemory(100 * constant.DefaultNvidiaGPUResourceMemory).
					Get()
				Expect(k8sClient.Create(ctx, &anotherNamespace)).To(Succeed())
				Expect(k8sClient.Create(ctx, &anotherElasticQuota)).To(Succeed())

				By("Creating a new Pod within the ElasticQuota namespace, requesting GPUMemory < EQ min")
				inQuotaPod := factory.BuildPod(elasticQuota.Namespace, "pod-1").
					WithContainer(
						factory.BuildContainer("container-1", "test:0.0.1").
							WithNvidiaGPULimit(inQuotaPodGPU).
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
							WithNvidiaGPULimit(overQuotaPodGPU).
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
				}, timeout, interval).Should(Equal(strconv.Itoa((inQuotaPodGPU + overQuotaPodGPU) * constant.DefaultNvidiaGPUResourceMemory)))

				By("Checking the in-quota Pod's capacity-info label is in-quota")
				assertPodHasLabel(
					ctx,
					inQuotaPod,
					v1alpha1.LabelCapacityInfo,
					string(constant.CapacityInfoInQuota),
					timeout,
					interval,
				)

				By("Checking the over-quota Pod's capacity-info label is over-quota")
				assertPodHasLabel(
					ctx,
					overQuotaPod,
					v1alpha1.LabelCapacityInfo,
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
					v1alpha1.LabelCapacityInfo,
					string(constant.CapacityInfoInQuota),
					timeout,
					interval,
				)
			})

			When("An ElasticQuota min field is updated", func() {
				It("Should update Pods capacity-info label accordingly", func() {
					const (
						elasticQuotaMinGPUMemory            = 4 * constant.DefaultNvidiaGPUResourceMemory
						elasticQuotaMinGPUMemoryAfterUpdate = 2 * constant.DefaultNvidiaGPUResourceMemory
						elasticQuotaMaxGPUMemory            = 6 * constant.DefaultNvidiaGPUResourceMemory

						podOneNvidiaGPU = 2
						podTwoNvidiaGPU = 2
					)

					By("Creating an ElasticQuota successfully")
					elasticQuota := v1alpha1.BuildEq(namespace.Name, util.RandomStringLowercase(10)).
						WithMinGPUMemory(elasticQuotaMinGPUMemory).
						WithMaxGPUMemory(elasticQuotaMaxGPUMemory).
						Get()
					Expect(k8sClient.Create(ctx, &elasticQuota)).To(Succeed())

					By("Creating new Pods within the ElasticQuota namespace, requesting total GPUMemory <= EQ min")
					podOne := factory.BuildPod(elasticQuota.Namespace, "pod-1").
						WithContainer(
							factory.BuildContainer("container-1", "test:0.0.1").
								WithNvidiaGPULimit(podOneNvidiaGPU).
								Get(),
						).
						Get()
					podTwo := factory.BuildPod(elasticQuota.Namespace, "pod-2").
						WithContainer(
							factory.BuildContainer("container-1", "test:0.0.1").
								WithNvidiaGPULimit(podTwoNvidiaGPU).
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
						v1alpha1.LabelCapacityInfo,
						string(constant.CapacityInfoInQuota),
						timeout,
						interval,
					)
					assertPodHasLabel(
						ctx,
						podTwo,
						v1alpha1.LabelCapacityInfo,
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
						v1alpha1.LabelCapacityInfo,
						string(constant.CapacityInfoInQuota),
						timeout,
						interval,
					)

					By("Checking that the Pod created last is now labelled as over-quota")
					assertPodHasLabel(
						ctx,
						podTwo,
						v1alpha1.LabelCapacityInfo,
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
