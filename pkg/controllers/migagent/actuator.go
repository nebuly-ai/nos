package migagent

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/migagent/types"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	migtypes "github.com/nebuly-ai/nebulnetes/pkg/gpu/mig/types"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
	"time"
)

type MigActuator struct {
	client.Client
	migClient mig.Client
	nodeName  string
	mutex     sync.Locker
}

func NewActuator(client client.Client, migClient mig.Client, mutex sync.Locker, nodeName string) MigActuator {
	return MigActuator{
		Client:    client,
		migClient: migClient,
		nodeName:  nodeName,
		mutex:     mutex,
	}
}

func (a *MigActuator) newLogger(ctx context.Context) klog.Logger {
	return log.FromContext(ctx).WithName("Actuator")
}

func (a *MigActuator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := a.newLogger(ctx)

	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Retrieve instance
	var instance v1.Node
	if err := a.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}

	// Check if reported status already matches spec
	statusAnnotations, specAnnotations := types.GetGPUAnnotationsFromNode(instance)
	if specMatchesStatus(specAnnotations, statusAnnotations) {
		logger.Info("reported status matches desired MIG config, nothing to do")
		return ctrl.Result{}, nil
	}

	// Compute MIG config plan
	plan, err := a.plan(ctx, specAnnotations)
	if err != nil {
		return ctrl.Result{}, err
	}
	if plan.IsEmpty() {
		logger.Info("MIG config plan is empty, nothing to do")
		return ctrl.Result{}, nil
	}

	// Apply MIG config plan
	return a.apply(ctx, plan)
}

func (a *MigActuator) plan(ctx context.Context, specAnnotations types.GPUSpecAnnotationList) (types.MigConfigPlan, error) {
	logger := a.newLogger(ctx)

	// Compute current state
	migDeviceResources, err := a.migClient.GetMigDeviceResources(ctx)
	if err != nil {
		logger.Error(err, "unable to get MIG device resources")
		return types.MigConfigPlan{}, err
	}
	state := types.NewMigState(migDeviceResources)

	// Check if actual state already matches spec
	if state.Matches(specAnnotations) {
		logger.Info("actual state matches desired MIG config")
		return types.MigConfigPlan{}, nil
	}

	// Compute MIG config plan
	return types.NewMigConfigPlan(state, specAnnotations), nil
}

func (a *MigActuator) apply(ctx context.Context, plan types.MigConfigPlan) (ctrl.Result, error) {
	var err error
	logger := a.newLogger(ctx)
	logger.Info(
		"applying MIG config plan",
		"createOperations",
		plan.CreateOperations,
		"deleteOperations",
		plan.DeleteOperations,
	)

	var atLeastOneDelete bool
	var atLeastOneCreate bool

	// Apply delete operations first
	for _, op := range plan.DeleteOperations {
		atLeastOneDelete, err = a.applyDeleteOp(ctx, op)
		if err != nil {
			logger.Error(err, "unable to fulfill delete operation", "op", op)
		}
	}

	// Apply create operations
	for _, op := range plan.CreateOperations {
		atLeastOneCreate, err = a.applyCreateOp(ctx, op)
		if err != nil {
			logger.Error(err, "unable to fulfill create operation", "op", op)
		}
	}

	// Restart nvidia device plugin to expose updated resources to k8s
	if atLeastOneCreate || atLeastOneDelete {
		if err = a.restartNvidiaDevicePlugin(ctx); err != nil {
			logger.Error(err, "unable to restart nvidia device plugin")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, err
}

// restartNvidiaDevicePlugin deletes the Nvidia Device Plugin pod and blocks until it is successfully recreated by
// its daemonset
func (a *MigActuator) restartNvidiaDevicePlugin(ctx context.Context) error {
	logger := a.newLogger(ctx)
	logger.Info("restarting nvidia device plugin")

	// delete pod on the current node
	var podList v1.PodList
	if err := a.Client.List(
		ctx,
		&podList,
		client.MatchingLabels{"app": "nvidia-device-plugin-daemonset"},
		client.MatchingFields{constant.PodNodeNameKey: a.nodeName},
	); err != nil {
		return err
	}
	if len(podList.Items) != 1 {
		return fmt.Errorf("error getting nvidia device plugin pod: expected exactly 1 but got %d", len(podList.Items))
	}
	if err := a.Client.Delete(ctx, &podList.Items[0]); err != nil {
		return fmt.Errorf("error deleting nvidia device plugin pod: %s", err.Error())
	}
	logger.V(1).Info("deleted nvidia device plugin pod")

	// wait for pod to restart
	if err := a.waitNvidiaDevicePluginPodRestart(ctx); err != nil {
		return err
	}
	logger.Info("nvidia device plugin restarted")

	return nil
}

func (a *MigActuator) waitNvidiaDevicePluginPodRestart(ctx context.Context) error {
	logger := a.newLogger(ctx)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	var podList v1.PodList
	checkPodRecreated := func() (bool, error) {
		if err := a.Client.List(
			ctx,
			&podList,
			client.MatchingLabels{"app": "nvidia-device-plugin-daemonset"},
			client.MatchingFields{constant.PodNodeNameKey: a.nodeName},
		); err != nil {
			return false, err
		}
		if len(podList.Items) != 1 {
			return false, nil
		}
		pod := podList.Items[0]
		if pod.DeletionTimestamp != nil {
			return false, nil
		}
		if pod.Status.Phase != v1.PodRunning {
			return false, nil
		}
		return true, nil
	}

	for {
		logger.V(1).Info("waiting for nvidia device plugin Pod to be recreated")
		recreated, err := checkPodRecreated()
		if err != nil {
			return err
		}
		if recreated {
			logger.V(1).Info("nvidia device plugin Pod recreated")
			break
		}
		if ctx.Err() != nil {
			return fmt.Errorf("error waiting for nvidia-device-plugin Pod on node %s: timeout", a.nodeName)
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}

func (a *MigActuator) applyDeleteOp(ctx context.Context, op types.DeleteOperation) (bool, error) {
	logger := a.newLogger(ctx)
	logger.Info("applying delete operation for MigProfile", "migProfile", op.MigProfile)

	// Get resources candidate to be deleted
	candidateResources := make([]migtypes.MigDeviceResource, 0)
	for _, r := range op.Resources {
		if r.Status == resource.StatusFree {
			logger.Info("resource added to delete candidates", "resource", r)
			candidateResources = append(candidateResources, r)
		}
		if r.Status != resource.StatusFree {
			logger.Info(
				"cannot add resource to delete candidates because status is not 'free'",
				"status",
				r.Status,
				"resource",
				r,
			)
		}
	}

	// Delete resources choosing from candidates
	var nDeleted uint8
	for _, r := range candidateResources {
		if err := a.migClient.DeleteMigResource(ctx, r); err != nil {
			logger.Error(err, "unable to delete MIG resource", "resource", r)
			continue
		}
		logger.Info("deleted MIG resource", "resource", r)
		nDeleted++
		if nDeleted >= op.Quantity {
			break
		}
	}

	atLeastOneDelete := nDeleted > 0

	// Return error if we couldn't delete the amount of resources specified by the Delete Operation
	if nDeleted < op.Quantity {
		return atLeastOneDelete, fmt.Errorf("could delete only %d out of %d MIG resources", nDeleted, op.Quantity)
	}

	return atLeastOneDelete, nil
}

func (a *MigActuator) applyCreateOp(ctx context.Context, op types.CreateOperation) (bool, error) {
	logger := a.newLogger(ctx)
	logger.Info("applying create operation for MigProfile", "migProfile", op.MigProfile)

	var nCreated uint8
	var i uint8
	for i = 0; i < op.Quantity; i++ {
		err := a.migClient.CreateMigResource(ctx, op.MigProfile)
		if err != nil {
			logger.Error(err, "unable to create MIG resource", "migProfile", op.MigProfile)
			continue
		}
		logger.Info("created MIG resource", "migProfile", op.MigProfile)
		nCreated++
	}

	atLeastOneCreate := nCreated > 0

	if nCreated < op.Quantity {
		return atLeastOneCreate, fmt.Errorf("could create only %d out of %d MIG resources", nCreated, op.Quantity)
	}

	return atLeastOneCreate, nil
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
