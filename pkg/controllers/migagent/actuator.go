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

type MIGActuator struct {
	client.Client
	migClient *mig.Client
}

func NewActuator(client client.Client, migClient *mig.Client) MIGActuator {
	reporter := MIGActuator{
		Client:    client,
		migClient: migClient,
	}
	return reporter
}

func (a *MIGActuator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("Actuator")
	logger.Info("Actuating desired MIG geometry")

	// Retrieve instance
	var instance v1.Node
	if err := a.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}

	//if specMatchesStatusAnnotations() {
	//
	//}

	return ctrl.Result{}, nil
}

func (a *MIGActuator) SetupWithManager(mgr ctrl.Manager, controllerName string, nodeName string) error {
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
