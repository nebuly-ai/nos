package node

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientretry "k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"time"
)

var updateNodeBackoff = wait.Backoff{
	Steps:    5,
	Duration: 100 * time.Millisecond,
	Jitter:   1.0,
}

type Provider struct {
	GetNode       func(ctx context.Context, nodeName string) (*v1.Node, error)
	GetNodeCached func(ctx context.Context, nodeName string) (*v1.Node, error)
}

func UpdateNodeAnnotations(ctx context.Context,
	k8sClient kubernetes.Interface,
	provider Provider,
	nodeName string,
	updateFunc func(annotations map[string]string)) error {

	logger := klog.FromContext(ctx)

	firstTry := true
	return clientretry.RetryOnConflict(updateNodeBackoff, func() error {
		var err error
		var node *v1.Node
		// First we try getting node from the API server cache, as it's cheaper. If it fails
		// we get it from etcd to be sure to have fresh data.
		if firstTry {
			node, err = provider.GetNodeCached(ctx, nodeName)
			firstTry = false
		} else {
			node, err = provider.GetNode(ctx, nodeName)
		}

		if err != nil {
			logger.Error(err, "unable to fetch node instance", "node", nodeName)
			return err
		}

		// Make a copy of the node and update the status annotations
		newNode := node.DeepCopy()
		if newNode.Annotations == nil {
			newNode.Annotations = make(map[string]string)
		}
		updateFunc(newNode.Annotations)

		// Patch node
		oldData, err := json.Marshal(node)
		if err != nil {
			return fmt.Errorf("failed to marshal the existing node %#v: %v", node, err)
		}
		newData, err := json.Marshal(newNode)
		if err != nil {
			return fmt.Errorf("failed to marshal the new node %#v: %v", newNode, err)
		}
		patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, &v1.Node{})
		if err != nil {
			return fmt.Errorf("failed to create a two-way merge patch: %v", err)
		}
		if _, err := k8sClient.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}); err != nil {
			return err
		}

		return nil
	})
}
