/*
 * Copyright 2022 Nebuly.ai
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package migagent

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/migagent/plan"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type MigActuator struct {
	client.Client
	migClient   mig.Client
	sharedState *SharedState
	nodeName    string

	// lastAppliedPlan is the latest applied plan
	lastAppliedPlan *plan.MigConfigPlan
	// lastAppliedStatus is the MIG status of the GPUs at the time when the latest plan was applied
	lastAppliedStatus *mig.GPUStatusAnnotationList
}

func NewActuator(client client.Client, migClient mig.Client, sharedState *SharedState, nodeName string) MigActuator {
	return MigActuator{
		Client:      client,
		migClient:   migClient,
		nodeName:    nodeName,
		sharedState: sharedState,
	}
}

func (a *MigActuator) newLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Actuator")
}

func (a *MigActuator) updateLastApplied(currentPlan plan.MigConfigPlan, currentStatus mig.GPUStatusAnnotationList) {
	a.lastAppliedPlan = &currentPlan
	a.lastAppliedStatus = &currentStatus
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch

func (a *MigActuator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := a.newLogger(ctx)

	// If we haven't reported the last applied config, requeue and avoid acquiring lock
	if !a.sharedState.AtLeastOneReportSinceLastApply() {
		logger.Info("last applied config hasn't been reported yet, waiting...")
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	a.sharedState.Lock()
	defer a.sharedState.Unlock()

	// Retrieve instance
	var instance v1.Node
	if err := a.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}

	// Check if reported status already matches spec
	statusAnnotations, specAnnotations := mig.GetGPUAnnotationsFromNode(instance)
	if mig.SpecMatchesStatus(specAnnotations, statusAnnotations) {
		logger.Info("reported status matches desired MIG config, nothing to do")
		return ctrl.Result{}, nil
	}

	// Compute MIG config plan
	configPlan, err := a.plan(ctx, specAnnotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	// At the end of reconcile, update last applied status information
	defer a.updateLastApplied(configPlan, statusAnnotations)

	// Check if plan has to be applied
	if configPlan.IsEmpty() {
		logger.Info("MIG config plan is empty, nothing to do")
		return ctrl.Result{}, nil
	}
	if configPlan.Equal(a.lastAppliedPlan) && statusAnnotations.Equal(a.lastAppliedStatus) {
		logger.Info("MIG config plan already applied and state hasn't changed, nothing to do")
		return ctrl.Result{}, nil
	}

	// Apply MIG config plan
	res, err := a.apply(ctx, configPlan)
	a.sharedState.OnApplyDone()

	return res, err
}

func (a *MigActuator) plan(ctx context.Context, specAnnotations mig.GPUSpecAnnotationList) (plan.MigConfigPlan, error) {
	logger := a.newLogger(ctx)

	// Compute current state
	migDeviceResources, err := a.migClient.GetMigDeviceResources(ctx)
	if gpu.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to get MIG device resources")
		return plan.MigConfigPlan{}, err
	}
	// If err is not found, restart the NVIDIA device plugin for updating the resources exposed to k8s
	if gpu.IsNotFound(err) {
		logger.Error(err, "unable to get MIG device resources")
		return plan.MigConfigPlan{}, a.restartNvidiaDevicePlugin(ctx)
	}

	state := plan.NewMigState(migDeviceResources)

	// Check if actual state already matches spec
	if state.Matches(specAnnotations) {
		logger.Info("actual state matches desired MIG config")
		return plan.MigConfigPlan{}, nil
	}

	// Compute MIG config plan
	return plan.NewMigConfigPlan(state, specAnnotations), nil
}

func (a *MigActuator) apply(ctx context.Context, plan plan.MigConfigPlan) (ctrl.Result, error) {
	logger := a.newLogger(ctx)
	logger.Info(
		"applying MIG config plan",
		"createOperations",
		plan.CreateOperations,
		"deleteOperations",
		plan.DeleteOperations,
	)

	var restartRequired bool
	var atLeastOneErr bool

	// Apply delete operations first
	for _, op := range plan.DeleteOperations {
		status := a.applyDeleteOp(ctx, op)
		if status.Err != nil {
			logger.Error(status.Err, "unable to fulfill delete operation", "op", op)
			atLeastOneErr = true
		}
		if status.PluginRestartRequired {
			restartRequired = true
		}
	}

	// Apply create operations
	status := a.applyCreateOps(ctx, plan.CreateOperations)
	if status.Err != nil {
		logger.Error(status.Err, "unable to fulfill create operations")
		atLeastOneErr = true
	}
	if status.PluginRestartRequired {
		restartRequired = true
	}

	// Restart the NVIDIA device plugin if necessary
	if restartRequired {
		if err := a.restartNvidiaDevicePlugin(ctx); err != nil {
			logger.Error(err, "unable to restart nvidia device plugin")
			return ctrl.Result{}, err
		}
	}

	// Check if any error happened
	if atLeastOneErr {
		return ctrl.Result{}, fmt.Errorf("at least one operation failed while applying desired MIG config")
	}

	return ctrl.Result{}, nil
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
	if err := a.waitNvidiaDevicePluginPodRestart(ctx, 1*time.Minute); err != nil {
		return err
	}
	logger.Info("nvidia device plugin restarted")

	return nil
}

func (a *MigActuator) waitNvidiaDevicePluginPodRestart(ctx context.Context, timeout time.Duration) error {
	logger := a.newLogger(ctx)

	ctx, cancel := context.WithTimeout(ctx, timeout)
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

func (a *MigActuator) applyDeleteOp(ctx context.Context, op plan.DeleteOperation) plan.OperationStatus {
	logger := a.newLogger(ctx)
	var restartRequired bool

	// Delete resources choosing from candidates
	var nDeleted int
	var deleteErrors = make(gpu.ErrorList, 0)
	for _, r := range op.Resources {
		if !r.IsFree() {
			err := fmt.Errorf("resource is not free")
			logger.Error(err, "cannot delete MIG resource", "resource", r)
			continue
		}
		err := a.migClient.DeleteMigResource(ctx, r)
		if gpu.IgnoreNotFound(err) != nil {
			deleteErrors = append(deleteErrors, err)
			logger.Error(err, "unable to delete MIG resource", "resource", r)
			continue
		}
		if gpu.IsNotFound(err) {
			deleteErrors = append(deleteErrors, err)
			logger.Error(err, "unable to delete MIG resource", "resource", r)
			restartRequired = true
			continue
		}
		logger.Info("deleted MIG resource", "resource", r)
		nDeleted++
	}

	if nDeleted > 0 {
		restartRequired = true
	}

	if len(deleteErrors) > 0 {
		return plan.OperationStatus{
			PluginRestartRequired: restartRequired,
			Err:                   deleteErrors,
		}
	}
	return plan.OperationStatus{
		PluginRestartRequired: restartRequired,
		Err:                   nil,
	}
}

func (a *MigActuator) applyCreateOps(ctx context.Context, ops plan.CreateOperationList) plan.OperationStatus {
	logger := a.newLogger(ctx)
	logger.Info("applying create operations", "migProfiles", ops)

	profileList := ops.Flatten()
	created, err := a.migClient.CreateMigResources(ctx, profileList)
	if err != nil {
		nCreated := len(created)
		return plan.OperationStatus{
			PluginRestartRequired: nCreated > 0,
			Err: fmt.Errorf(
				"could create only %d out of %d MIG resources: %s",
				nCreated,
				len(profileList),
				err,
			),
		}
	}
	return plan.OperationStatus{
		PluginRestartRequired: true,
		Err:                   nil,
	}
}

func (a *MigActuator) SetupWithManager(mgr ctrl.Manager, controllerName string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1.Node{},
			builder.WithPredicates(
				excludeDeletePredicate{},
				matchingNamePredicate{Name: a.nodeName},
				annotationsChangedPredicate{},
			),
		).
		Named(controllerName).
		Complete(a)
}
