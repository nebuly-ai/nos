package mighandler

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"time"
)

type MIGActuator struct {
	k8sClient kubernetes.Interface
	migClient *mig.Client
	logger    klog.Logger

	node            string
	refreshInterval time.Duration
}

func NewMIGActuator(node string, k8sClient kubernetes.Interface, migClient *mig.Client, sharedFactory informers.SharedInformerFactory, refreshInterval time.Duration) MIGActuator {
	nodeInformer := sharedFactory.Core().V1().Nodes().Informer()
	actuator := MIGActuator{
		refreshInterval: refreshInterval,
		migClient:       migClient,
		k8sClient:       k8sClient,
		logger:          klog.NewKlogr().WithName("actuator"),
	}
	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    actuator.nodeAdded,
		UpdateFunc: actuator.nodeUpdated,
	})
	return actuator
}

func (r *MIGActuator) nodeUpdated(_, newObj interface{}) {
	r.logger.Info("node updated")
	//newNode := newObj.(*v1.Node)
	//newGPUStatusAnnotations := getStatusAnnotations(newNode)
}

func (r *MIGActuator) nodeAdded(node interface{}) {
	r.logger.Info("node added")
}

// Start starts running the MIGActuator. The reporter will stop running
// when the context is closed. Start blocks until the context is closed.
func (r *MIGActuator) Start(ctx context.Context) error {

	// Start informers
	//go r.nodeInformer.Informer().Run(ctx.Done())
	//logger.Info("Waiting for shared informer cached sync...")
	//if !cache.WaitForCacheSync(ctx.Done(), r.nodeInformer.Informer().HasSynced) {
	//	return fmt.Errorf("timed out waiting for caches to sync")
	//}
	//logger.Info("Informer cache synced")

	// Schedule refresh
	ticker := time.NewTicker(r.refreshInterval)
	for {
		select {
		case <-ticker.C:
			//if err := r.ReportMIGResourcesStatus(ctx); err != nil {
			//	logger.Error(err, "unable to report MIG status")
			//}
		case <-ctx.Done():
			r.logger.V(3).Info("ctx done: stop actuating MIG geometry")
			return nil
		}
	}
}
