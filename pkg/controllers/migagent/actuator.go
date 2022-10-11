package migagent

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

type MIGActuator struct {
	k8sClient kubernetes.Interface
	migClient *mig.Client
	logger    klog.Logger

	node            string
	refreshInterval time.Duration
}

func (a *MIGActuator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
