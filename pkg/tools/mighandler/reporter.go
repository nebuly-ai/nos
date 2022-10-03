package mighandler

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"time"
)

type MIGReporter struct {
	k8sClient                *kubernetes.Clientset
	nodeInformer             informersv1.NodeInformer
	node                     string
	podResourcesListerClient pdrv1.PodResourcesListerClient
	refreshInterval          time.Duration
}

func NewMIGReporter(node string, k8sClient *kubernetes.Clientset, sharedFactory informers.SharedInformerFactory, client pdrv1.PodResourcesListerClient, refreshInterval time.Duration) MIGReporter {
	nodeInformer := sharedFactory.Core().V1().Nodes()
	reporter := MIGReporter{
		k8sClient:                k8sClient,
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
	// 1. compute MIG geometry
	// 2. convert MIG geometry to annotations
	// 3. update node annotations
	//logger := klog.FromContext(ctx)

	return nil
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
