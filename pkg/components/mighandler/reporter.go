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
	"reflect"
	"strings"
	"time"
)

type MIGReporter struct {
	k8sClient    kubernetes.Interface
	migClient    *mig.Client
	nodeProvider nodeutil.Provider

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
	reporter.nodeProvider = nodeutil.Provider{
		GetNode: func(ctx context.Context, nodeName string) (*v1.Node, error) {
			return reporter.k8sClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		},
		GetNodeCached: func(_ context.Context, nodeName string) (*v1.Node, error) {
			return reporter.nodeInformer.Lister().Get(nodeName)
		},
	}
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
	logger.Info("Reporting MIG resources status", "node", r.node)

	// Compute new status annotations
	usedMIGs, err := r.migClient.GetUsedMIGDevices(ctx)
	logger.V(3).Info("Loaded used MIG devices", "usedMIGs", usedMIGs)
	if err != nil {
		return err
	}
	freeMIGs, err := r.migClient.GetFreeMIGDevices(ctx)
	logger.V(3).Info("Loaded free MIG devices", "freeMIGs", usedMIGs)
	if err != nil {
		return err
	}
	newStatusAnnotations := computeStatusAnnotations(usedMIGs, freeMIGs)

	// Get current status annotations and compare with new ones
	oldStatusAnnotations, err := r.readCurrentStatusAnnotations()
	if err != nil {
		logger.Error(err, "Unable to read current status annotations from node")
		return err
	}
	if reflect.DeepEqual(newStatusAnnotations, oldStatusAnnotations) {
		logger.Info("Current status is equal to last reported status, nothing to do")
		return nil
	}

	// Update node
	logger.Info("Status changed - reporting it by updating node annotations")
	updateFunc := func(annotations map[string]string) {
		for k := range annotations {
			if strings.HasPrefix(k, v1alpha1.AnnotationGPUStatusPrefix) {
				delete(annotations, k)
			}
		}
		for k, v := range newStatusAnnotations {
			annotations[k] = v
		}
	}
	err = nodeutil.UpdateNodeAnnotations(
		ctx,
		r.k8sClient,
		r.nodeProvider,
		r.node,
		updateFunc,
	)
	if err != nil {
		logger.Error(err, "unable to update node status annotations")
		return err
	}
	logger.Info("Updated status reported - node annotations updated successfully")

	return nil
}

func (r *MIGReporter) readCurrentStatusAnnotations() (map[string]string, error) {
	node, err := r.nodeInformer.Lister().Get(r.node)
	if err != nil {
		return nil, err
	}
	res := make(map[string]string)
	for k, v := range node.Annotations {
		if strings.HasPrefix(k, v1alpha1.AnnotationGPUStatusPrefix) {
			res[k] = v
		}
	}
	return res, nil
}

func computeStatusAnnotations(used []mig.Device, free []mig.Device) map[string]string {
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
			ticker.Stop()
			return
		}
	}()
	return nil
}
