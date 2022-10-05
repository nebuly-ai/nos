package mighandler

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	nodeutil "github.com/nebuly-ai/nebulnetes/pkg/util/node"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"strings"
	"time"
)

type MIGReporter struct {
	k8sClient kubernetes.Interface
	migClient *mig.Client

	nodeInformer    informersv1.NodeInformer
	node            string
	refreshInterval time.Duration
}

func NewMIGReporter(node string, k8sClient kubernetes.Interface, migClient *mig.Client, sharedFactory informers.SharedInformerFactory, refreshInterval time.Duration) MIGReporter {
	nodeInformer := sharedFactory.Core().V1().Nodes()
	reporter := MIGReporter{
		k8sClient:       k8sClient,
		migClient:       migClient,
		nodeInformer:    nodeInformer,
		refreshInterval: refreshInterval,
		node:            node,
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
	if err := r.ReportMIGResourcesStatus(context.Background()); err != nil {
		klog.Error("unable to report MIG status", err)
	}
}

func (r *MIGReporter) nodeUpdated(old, newObj interface{}) {
	oldNode := old.(*v1.Node)
	newNode := newObj.(*v1.Node)
	if !equality.Semantic.DeepEqual(oldNode.Status.Allocatable, newNode.Status.Allocatable) {
		if err := r.ReportMIGResourcesStatus(context.Background()); err != nil {
			klog.Error("unable to report MIG status", err)
		}
	}
}

func (r *MIGReporter) ReportMIGResourcesStatus(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	// Compute new status annotations
	usedMIGs, err := r.migClient.GetUsedMIGDevices(ctx)
	if err != nil {
		return err
	}
	freeMIGs, err := r.migClient.GetFreeMIGDevices(ctx)
	if err != nil {
		return err
	}
	statusAnnotations := getStatusAnnotations(usedMIGs, freeMIGs)

	// Update node
	updateFunc := func(annotations map[string]string) {
		for k := range annotations {
			if strings.HasPrefix(k, v1alpha1.AnnotationGPUStatusPrefix) {
				delete(annotations, k)
			}
		}
		for k, v := range statusAnnotations {
			annotations[k] = v
		}
	}
	nodeProvider := nodeutil.Provider{
		GetNode: func(ctx context.Context, nodeName string) (*v1.Node, error) {
			return r.k8sClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		},
		GetNodeCached: func(_ context.Context, nodeName string) (*v1.Node, error) {
			return r.nodeInformer.Lister().Get(nodeName)
		},
	}
	err = nodeutil.UpdateNodeAnnotations(
		ctx,
		r.k8sClient,
		nodeProvider,
		r.node,
		updateFunc,
	)

	if err != nil {
		logger.Error(err, "unable to update node status annotations", "node", r.node)
		return err
	}

	return nil
}

func getStatusAnnotations(used []mig.Device, free []mig.Device) map[string]string {
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
		key := fmt.Sprintf(v1alpha1.AnnotationUsedMIGStatusFormat, u.GpuIndex, u.ResourceName)
		res[key] = fmt.Sprintf("%d", quantity)
	}
	// Free annotations
	for _, u := range free {
		quantity, _ := freeMigToQuantity[u.FullResourceName()]
		key := fmt.Sprintf(v1alpha1.AnnotationFreeMIGStatusFormat, u.GpuIndex, u.ResourceName)
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
			if err := r.ReportMIGResourcesStatus(ctx); err != nil {
				logger.Error(err, "unable to report MIG status")
			}
		case <-ctx.Done():
			return
		}
	}()

	return nil
}
