//go:build integration

package migagent

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	migtypes "github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var _ = Describe("MigAgent - Reporter", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	BeforeEach(func() {
		// Reset mig client
		reporterMigClient.Reset()

		// Fetch the node and clean-up annotations
		var node v1.Node
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: reporterNodeName, Namespace: ""}, &node)).To(Succeed())
		updated := node.DeepCopy()
		updated.Annotations = map[string]string{}
		Expect(k8sClient.Patch(ctx, updated, client.MergeFrom(&node))).To(Succeed())
	})

	AfterEach(func() {
	})

	When("New MIG resources are created on the node", func() {
		It("Should expose those resources on the node as annotations", func() {
			By("Fetching the node")
			var node v1.Node
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: reporterNodeName, Namespace: ""}, &node)).To(Succeed())

			By("Checking the node does not have any annotation")
			Expect(node.Annotations).To(BeEmpty())

			By("Checking that after some time the node will have the annotations exposing the new resources")
			reporterMigClient.ReturnedMigDeviceResources = []migtypes.MigDeviceResource{
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-1g.10gb",
						DeviceId:     "id-1",
						Status:       "free",
					},
					GpuIndex: 0,
				},
				{
					Device: resource.Device{
						ResourceName: "nvidia.com/mig-2g.20gb",
						DeviceId:     "id-2",
						Status:       "used",
					},
					GpuIndex: 1,
				},
			}
			expectedAnnotationOne := fmt.Sprintf(v1alpha1.AnnotationFreeMigStatusFormat, 0, "1g.10gb")
			expectedAnnotationTwo := fmt.Sprintf(v1alpha1.AnnotationUsedMigStatusFormat, 1, "2g.20gb")
			expectedAnnotations := map[string]string{
				expectedAnnotationOne: "1",
				expectedAnnotationTwo: "1",
			}
			Eventually(func() map[string]string {
				var updatedNode v1.Node
				err := k8sClient.Get(ctx, types.NamespacedName{Name: reporterNodeName, Namespace: ""}, &updatedNode)
				if err != nil {
					return nil
				}
				return updatedNode.Annotations
			}, timeout, interval).Should(Equal(expectedAnnotations))
		})
	})
})
