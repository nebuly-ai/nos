package mighandler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	clientretry "k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"strings"
	"time"
)

type MIGReporter struct {
	k8sClient kubernetes.Interface
	gpuClient *gpu.Client

	nodeInformer             informersv1.NodeInformer
	node                     string
	podResourcesListerClient pdrv1.PodResourcesListerClient
	refreshInterval          time.Duration
}

func NewMIGReporter(node string, k8sClient kubernetes.Interface, gpuClient *gpu.Client, sharedFactory informers.SharedInformerFactory, client pdrv1.PodResourcesListerClient, refreshInterval time.Duration) MIGReporter {
	nodeInformer := sharedFactory.Core().V1().Nodes()
	reporter := MIGReporter{
		k8sClient:                k8sClient,
		gpuClient:                gpuClient,
		nodeInformer:             nodeInformer,
		podResourcesListerClient: client,
		refreshInterval:          refreshInterval,
		node:                     node,
	}
	nodeInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    reporter.nodeAdded,
			UpdateFunc: reporter.nodeUpdated,
		},
	)
	return reporter
}

func (r *MIGReporter) nodeAdded(_ interface{}) {
	if err := r.ReportMIGStatus(context.Background()); err != nil {
		klog.Error("unable to report MIG status", err)
	}
}

func (r *MIGReporter) nodeUpdated(old, newObj interface{}) {
	oldNode := old.(*v1.Node)
	newNode := newObj.(*v1.Node)
	if !equality.Semantic.DeepEqual(oldNode.Status.Allocatable, newNode.Status.Allocatable) {
		if err := r.ReportMIGStatus(context.Background()); err != nil {
			klog.Error("unable to report MIG status", err)
		}
	}
}

func (r *MIGReporter) ReportMIGStatus(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	// Compute new status annotations
	usedMIGs, err := r.gpuClient.GetUsedMIGDevices(ctx)
	if err != nil {
		return err
	}
	freeMIGs, err := r.gpuClient.GetFreeMIGDevices(ctx)
	if err != nil {
		return err
	}
	statusAnnotations := getStatusAnnotations(usedMIGs, freeMIGs)

	// Update node
	firstTry := false
	var backoff = wait.Backoff{
		Steps:    5,
		Duration: 100 * time.Millisecond,
		Jitter:   1.0,
	}
	err = clientretry.RetryOnConflict(backoff, func() error {
		var err error
		var node *v1.Node
		// First we try getting node from the API server cache, as it's cheaper. If it fails
		// we get it from etcd to be sure to have fresh data.
		// Fetch watched node
		if firstTry {
			node, err = r.nodeInformer.Lister().Get(r.node)
			firstTry = false
		} else {
			node, err = r.k8sClient.CoreV1().Nodes().Get(ctx, r.node, metav1.GetOptions{})
		}

		if err != nil {
			logger.Error(err, "unable to fetch node instance", "node", r.node)
		}
		if err != nil {
			return err
		}

		// Make a copy of the node and update the status annotations
		newNode := node.DeepCopy()
		if newNode.Annotations == nil {
			newNode.Annotations = make(map[string]string)
		}
		for k := range newNode.Annotations {
			if strings.HasPrefix(k, "n8s.nebuly.ai/status/gpu") {
				delete(newNode.Annotations, k)
			}
		}
		for k, v := range statusAnnotations {
			newNode.Annotations[k] = v
		}

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
		if _, err := r.k8sClient.CoreV1().Nodes().Patch(context.TODO(), node.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		logger.Error(err, "unable to update node status annotations", "node", r.node)
		return err
	}

	return nil
}

func getStatusAnnotations(used []gpu.MIGDevice, free []gpu.MIGDevice) map[string]string {
	res := make(map[string]string)

	// Compute used MIG devices quantities
	usedMigToQuantity := make(map[string]int)
	for _, u := range used {
		key := u.FullResourceName()
		currentCount, _ := usedMigToQuantity[key]
		currentCount++
		usedMigToQuantity[key] = currentCount
	}
	// Compute free MIG devices quantities
	freeMigToQuantity := make(map[string]int)
	for _, u := range free {
		key := u.FullResourceName()
		currentCount, _ := freeMigToQuantity[key]
		currentCount++
		freeMigToQuantity[key] = currentCount
	}

	// Used annotations
	for _, u := range used {
		quantity, _ := usedMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf("n8s.nebuly.ai/status/gpu/%d/%s/used", u.GpuIndex, u.ResourceName)
		res[key] = fmt.Sprintf("%d", quantity)
	}
	// Free annotations
	for _, u := range free {
		quantity, _ := freeMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf("n8s.nebuly.ai/status/gpu/%d/%s/free", u.GpuIndex, u.ResourceName)
		res[key] = fmt.Sprintf("%d", quantity)
	}

	return res
}

func (r *MIGReporter) Start(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	// Start informers
	go r.nodeInformer.Informer().Run(ctx.Done())
	logger.Info("Waiting for shared informer cached sync...")
	if !cache.WaitForCacheSync(ctx.Done(), r.nodeInformer.Informer().HasSynced) {
		return fmt.Errorf("timed out waiting for caches to sync")
	}
	logger.Info("Informer cache synced")

	// Schedule refresh
	ticker := time.NewTicker(r.refreshInterval)
	go func() {
		select {
		case <-ticker.C:
			if err := r.ReportMIGStatus(ctx); err != nil {
				logger.Error(err, "unable to report MIG status")
			}
		case <-ctx.Done():
			return
		}
	}()

	return nil
}
