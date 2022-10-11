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

	logger klog.Logger
}

func NewMIGReporter(node string, k8sClient kubernetes.Interface, migClient *mig.Client, sharedFactory informers.SharedInformerFactory, refreshInterval time.Duration) MIGReporter {
	nodeInformer := sharedFactory.Core().V1().Nodes()
	reporter := MIGReporter{
		k8sClient:       k8sClient,
		migClient:       migClient,
		nodeInformer:    nodeInformer,
		refreshInterval: refreshInterval,
		node:            node,
		logger:          klog.NewKlogr().WithName("reporter"),
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
	r.logger.Info("node added")
	if err := r.ReportMIGResourcesStatus(context.Background()); err != nil {
		klog.Error("unable to report MIG status", err)
	}
}

func (r *MIGReporter) nodeUpdated(old, newObj interface{}) {
	r.logger.Info("node updated")
	oldNode := old.(*v1.Node)
	newNode := newObj.(*v1.Node)
	if equality.Semantic.DeepEqual(oldNode.Status.Allocatable, newNode.Status.Allocatable) {
		klog.Info("allocatable resources unchanged, nothing to do")
		return
	}
	klog.Info("allocatable resources changed")
	if err := r.ReportMIGResourcesStatus(context.Background()); err != nil {
		klog.Error("unable to report MIG status", err)
	}
}

func (r *MIGReporter) ReportMIGResourcesStatus(ctx context.Context) error {
	r.logger.Info("Reporting MIG resources status", "node", r.node)

	// Compute new status annotations
	usedMIGs, err := r.migClient.GetUsedMIGDevices(ctx)
	r.logger.V(3).Info("Loaded used MIG devices", "usedMIGs", usedMIGs)
	if err != nil {
		return err
	}
	freeMIGs, err := r.migClient.GetFreeMIGDevices(ctx)
	r.logger.V(3).Info("Loaded free MIG devices", "freeMIGs", usedMIGs)
	if err != nil {
		return err
	}
	newStatusAnnotations := computeStatusAnnotations(usedMIGs, freeMIGs)

	// Get current status annotations and compare with new ones
	oldStatusAnnotations, err := r.readCurrentStatusAnnotations()
	if err != nil {
		r.logger.Error(err, "Unable to read current status annotations from node")
		return err
	}
	if reflect.DeepEqual(newStatusAnnotations, oldStatusAnnotations) {
		r.logger.Info("Current status is equal to last reported status, nothing to do")
		return nil
	}

	// Update node
	r.logger.Info("Status changed - reporting it by updating node annotations")
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
		r.logger.Error(err, "unable to update node status annotations")
		return err
	}
	r.logger.Info("Updated status reported - node annotations updated successfully")

	return nil
}

func (r *MIGReporter) readCurrentStatusAnnotations() (map[string]string, error) {
	node, err := r.nodeInformer.Lister().Get(r.node)
	if err != nil {
		return nil, err
	}
	return getStatusAnnotations(node), nil
}

// Start starts running the MIGReporter. The reporter will stop running
// when the context is closed. Start blocks until the context is closed.
func (r *MIGReporter) Start(ctx context.Context) error {
	// Start informers
	go r.nodeInformer.Informer().Run(ctx.Done())
	r.logger.Info("Waiting for shared informer cached sync...")
	if !cache.WaitForCacheSync(ctx.Done(), r.nodeInformer.Informer().HasSynced) {
		return fmt.Errorf("timed out waiting for caches to sync")
	}
	r.logger.Info("Informer cache synced")

	// Schedule refresh
	ticker := time.NewTicker(r.refreshInterval)
	for {
		select {
		case <-ticker.C:
			if err := r.ReportMIGResourcesStatus(ctx); err != nil {
				r.logger.Error(err, "unable to report MIG status")
			}
		case <-ctx.Done():
			r.logger.V(3).Info("ctx done: stop reporting MIG resources status")
			return nil
		}
	}
}
