package migagent

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type MigActuator struct {
	client.Client
	migClient *mig.Client
}

func NewActuator(client client.Client, migClient *mig.Client) MigActuator {
	reporter := MigActuator{
		Client:    client,
		migClient: migClient,
	}
	return reporter
}

func (a *MigActuator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("Actuator")
	logger.Info("Actuating desired MIG config")

	// Retrieve instance
	var instance v1.Node
	if err := a.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}

	// Check if status already matches spec
	if specMatchesStatusAnnotations(instance) {
		logger.Info("Status matches desired MIG config, nothing to do")
		return ctrl.Result{}, nil
	}

	getStatusAnnotations(instance)
	getSpecAnnotations(instance)

	return ctrl.Result{}, nil
}

func (a *MigActuator) SetupWithManager(mgr ctrl.Manager, controllerName string, nodeName string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1.Node{},
			builder.WithPredicates(
				excludeDeletePredicate{},
				matchingNamePredicate{Name: nodeName},
				annotationsChangedPredicate{},
			),
		).
		Named(controllerName).
		Complete(a)
}
