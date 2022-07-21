/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/constants"
	"github.com/nebuly-ai/nebulnetes/controllers/components"
	"github.com/nebuly-ai/nebulnetes/controllers/reconcilers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ModelDeploymentReconciler reconciles a ModelDeployment object
type ModelDeploymentReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
	ModelLibrary  components.ModelLibrary
}

func (r *ModelDeploymentReconciler) updateStatus(ctx context.Context, instance *n8sv1alpha1.ModelDeployment, logger logr.Logger) {
	var currentModelDeployment n8sv1alpha1.ModelDeployment
	namespacedName := types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}

	if err := r.Get(ctx, namespacedName, &currentModelDeployment); err != nil {
		logger.Error(err, "unable to fetch ModelDeployment")
		return
	}
	if equality.Semantic.DeepEqual(currentModelDeployment.Status, instance.Status) {
		logger.V(1).Info("current status and desired status of ModelDeployment are equal, skipping update")
		return
	}
	logger.Info("updating ModelDeployment status", "Status", instance.Status)
	if err := r.Status().Update(ctx, instance); err != nil {
		logger.Error(err, "unable to update ModelDeployment status")
	}
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=create;get;list;watch;delete

func (r *ModelDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var res ctrl.Result

	// Fetch ModelDeployment
	var instance = new(n8sv1alpha1.ModelDeployment)
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		logger.Error(err, "unable to fetch ModelDeployment")
		return ctrl.Result{}, err
	}
	loader := components.NewModelDeploymentComponentLoader(r.Client, instance)

	// Setup reconcilers
	analysisJobReconciler, err := reconcilers.NewAnalysisJobReconciler(
		r.Client,
		r.Scheme,
		r.EventRecorder,
		r.ModelLibrary,
		loader,
		instance,
	)
	if err != nil {
		logger.Error(err, "unable to create analysis job reconciler")
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, constants.EventInternalError, err.Error())
		instance.Status.State = n8sv1alpha1.StatusStateFailed
		r.updateStatus(ctx, instance, logger)
		return ctrl.Result{}, err
	}
	optimizationJobReconciler, err := reconcilers.NewOptimizationJobReconciler(
		r.Client,
		r.Scheme,
		r.EventRecorder,
		r.ModelLibrary,
		loader,
		instance,
	)
	if err != nil {
		logger.Error(err, "unable to create optimization job reconciler")
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, constants.EventInternalError, err.Error())
		instance.Status.State = n8sv1alpha1.StatusStateFailed
		r.updateStatus(ctx, instance, logger)
		return ctrl.Result{}, err
	}
	inferenceServiceReconciler := reconcilers.NewInferenceServiceReconciler(
		r.Client,
		r.Scheme,
		r.EventRecorder,
		r.ModelLibrary,
		loader,
		instance,
	)

	// Reconcile
	reconcilerChain := []components.ComponentReconciler{
		analysisJobReconciler,
		optimizationJobReconciler,
		inferenceServiceReconciler,
	}
	for _, reconciler := range reconcilerChain {
		res, err = reconciler.Reconcile(ctx)
		if err != nil {
			instance.Status.State = n8sv1alpha1.StatusStateFailed
			r.updateStatus(ctx, instance, logger)
			return ctrl.Result{}, err
		}
		if res.Requeue || res.RequeueAfter > 0 {
			return res, nil
		}
	}

	r.updateStatus(ctx, instance, logger)
	return res, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelDeploymentReconciler) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&n8sv1alpha1.ModelDeployment{}).
		Owns(&batchv1.Job{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
