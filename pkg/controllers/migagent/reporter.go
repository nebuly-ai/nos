package migagent

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
	"time"
)

type MIGReporter struct {
	client.Client
	migClient       *mig.Client
	refreshInterval time.Duration
}

func NewReporter(client client.Client, migClient *mig.Client, refreshInterval time.Duration) MIGReporter {
	reporter := MIGReporter{
		Client:          client,
		migClient:       migClient,
		refreshInterval: refreshInterval,
	}
	return reporter
}

func (r *MIGReporter) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := klog.FromContext(ctx)
	logger.Info("Reconciling MIG resources status")

	var instance v1.Node
	if err := r.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}

	// Compute new status annotations
	usedMIGs, err := r.migClient.GetUsedMIGDevices(ctx)
	if err != nil {
		logger.Error(err, "unable to get used MIG devices")
		return ctrl.Result{}, err
	}
	freeMIGs, err := r.migClient.GetFreeMIGDevices(ctx)
	if err != nil {
		logger.Error(err, "unable to get free MIG devices")
		return ctrl.Result{}, err
	}
	logger.V(3).Info("Loaded free MIG devices", "freeMIGs", usedMIGs)
	logger.V(3).Info("Loaded used MIG devices", "usedMIGs", usedMIGs)
	newStatusAnnotations := computeStatusAnnotations(usedMIGs, freeMIGs)

	// Get current status annotations and compare with new ones
	oldStatusAnnotations := getStatusAnnotations(instance)
	if reflect.DeepEqual(newStatusAnnotations, oldStatusAnnotations) {
		logger.Info("Current status is equal to last reported status, nothing to do")
		return ctrl.Result{RequeueAfter: r.refreshInterval}, nil
	}

	// Update node
	logger.Info("Status changed - reporting it by updating node annotations")
	updated := instance.DeepCopy()
	for k := range updated.Annotations {
		if strings.HasPrefix(k, v1alpha1.AnnotationGPUStatusPrefix) {
			delete(updated.Annotations, k)
		}
	}
	for k, v := range newStatusAnnotations {
		updated.Annotations[k] = v
	}
	if err := r.Client.Patch(ctx, updated, client.MergeFrom(&instance)); err != nil {
		logger.Error(err, "unable to update node status annotations")
		return ctrl.Result{}, err
	}

	logger.Info("Updated reported status - node annotations updated successfully")
	return ctrl.Result{RequeueAfter: r.refreshInterval}, nil
}

func (r *MIGReporter) SetupWithManager(mgr ctrl.Manager, controllerName string, nodeName string) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&source.Kind{Type: &v1.Node{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(
				matchingNamePredicate{Name: nodeName},
				nodeResourcesChangedPredicate{},
			),
		).
		Named(controllerName).
		Complete(r)
}
