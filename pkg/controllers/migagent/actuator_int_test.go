//go:build integration

package migagent

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("MigAgent - Actuator", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	BeforeEach(func() {
		// Reset mig client
		actuatorMigClient.Reset()

		// Fetch the node and clean-up annotations
		var node v1.Node
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: actuatorNodeName, Namespace: ""}, &node)).To(Succeed())
		updated := node.DeepCopy()
		updated.Annotations = map[string]string{}
		Expect(k8sClient.Patch(ctx, updated, client.MergeFrom(&node))).To(Succeed())

		fmt.Println("************ before each executed **************")
	})

	AfterEach(func() {
	})

	When("The node annotation gets updated with new GPU specifications", func() {
		It("Should create the extra mig profiles and restart the nvidia-device-plugin pod", func() {
			By("Fetching the node")
			var node v1.Node
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: actuatorNodeName, Namespace: ""}, &node)).To(Succeed())

			By("Fetching the nvidia-device-plugin pod")
			var nvidiaDevicePluginPod v1.Pod
			Expect(
				k8sClient.Get(
					ctx,
					types.NamespacedName{Name: actuatorNvidiaDevicePluginPodName, Namespace: nvidiaDevicePluginPodNamespace},
					&nvidiaDevicePluginPod),
			).To(Succeed())

			By("Updating the node annotations")
			annotation := fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.10gb")
			updatedNode := node.DeepCopy()
			updatedNode.Annotations = map[string]string{
				annotation: "1",
			}
			Expect(k8sClient.Patch(ctx, updatedNode, client.MergeFrom(&node)))

			By("Deleting the nvidia-device-plugin Pod on the node")
			Eventually(func() bool {
				var updatedPod v1.Pod
				err := k8sClient.Get(
					ctx,
					types.NamespacedName{Name: actuatorNvidiaDevicePluginPodName, Namespace: nvidiaDevicePluginPodNamespace},
					&updatedPod,
				)
				if client.IgnoreNotFound(err) != nil {
					return false
				}
				return updatedPod.DeletionTimestamp != nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	When("The node annotation gets updated but the spec and status don't change", func() {
		It("Should do nothing", func() {
			By("Fetching the node")
			var node v1.Node
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: actuatorNodeName, Namespace: ""}, &node)).To(Succeed())

			By("Fetching the nvidia-device-plugin pod")
			var nvidiaDevicePluginPod v1.Pod
			Expect(
				k8sClient.Get(
					ctx,
					types.NamespacedName{Name: actuatorNvidiaDevicePluginPodName, Namespace: nvidiaDevicePluginPodNamespace},
					&nvidiaDevicePluginPod),
			).To(Succeed())

			By("Updating the node annotations")
			specAnnotation := fmt.Sprintf(v1alpha1.AnnotationGPUMigSpecFormat, 0, "1g.10gb")
			statusAnnotation := fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, "1g.10gb")
			updatedNode := node.DeepCopy()
			updatedNode.Annotations = map[string]string{
				specAnnotation:   "1",
				statusAnnotation: "1",
			}
			Expect(k8sClient.Patch(ctx, updatedNode, client.MergeFrom(&node)))

			By("Do not deleting nvidia-device-plugin pod")
			Consistently(func() error {
				return k8sClient.Get(
					ctx,
					types.NamespacedName{Name: actuatorNvidiaDevicePluginPodName, Namespace: nvidiaDevicePluginPodNamespace},
					&v1.Pod{},
				)
			}, 5*time.Second).Should(Succeed())

			By("Do not calling the delete method on the MIG client")
			Expect(actuatorMigClient.NumCallsDeleteMigResource).To(BeZero())

			By("Do not calling the create method on the MIG client")
			Expect(actuatorMigClient.NumCallsCreateMigResource).To(BeZero())
		})
	})
})
