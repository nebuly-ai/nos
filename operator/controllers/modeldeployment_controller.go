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
	"fmt"
	"github.com/go-logr/logr"
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/constants"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// Format used for generating the names of the optimization jobs
	optimizationJobNameFormat = "%s-optimization"
	// Name of the controller of ModelDeployment kind
	modelDeploymentControllerName = "modeldeployment-controller"
)

// ModelDeploymentReconciler reconciles a ModelDeployment object
type ModelDeploymentReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
	ModelLibrary  ModelLibrary
}

type components struct {
	optimizationJob *batchv1.Job
}

func constructModelOptimizerContainer(ml ModelLibrary, md *n8sv1alpha1.ModelDeployment) (*corev1.Container, error) {
	mlCredentials, err := ml.GetCredentials()
	if err != nil {
		return nil, err
	}

	envVars := make([]corev1.EnvVar, 0)
	for key, value := range mlCredentials {
		envVars = append(envVars, corev1.EnvVar{Name: key, Value: value})
	}
	modelOptimizerImage := fmt.Sprintf(
		"%s:%s",
		md.Spec.Optimization.ModelOptimizerImageName,
		md.Spec.Optimization.ModelOptimizerImageVersion,
	)

	return &corev1.Container{
		Name:                     "optimizer",
		Image:                    modelOptimizerImage,
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		Env:                      envVars,
		Args: []string{
			md.Spec.SourceModel.Uri,
			ml.GetBaseUri(md),
			ml.GetOptimizedModelDescriptorUri(md),
			string(ml.GetStorageKind()),
			string(md.Spec.Optimization.Target),
		},
	}, nil
}

func (r *ModelDeploymentReconciler) buildOptimizationJob(instance *n8sv1alpha1.ModelDeployment) (*batchv1.Job, error) {
	container, err := constructModelOptimizerContainer(r.ModelLibrary, instance)
	if err != nil {
		return nil, err
	}

	optimizationJobBackoffLimit := int32(instance.Spec.Optimization.OptimizationJobBackoffLimit)
	var runAsNonRoot = true

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        fmt.Sprintf(optimizationJobNameFormat, instance.Name),
			Namespace:   instance.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &optimizationJobBackoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: &runAsNonRoot},
					Containers:      []corev1.Container{*container},
					RestartPolicy:   corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Set labels
	job.Labels[constants.LabelCreatedBy] = modelDeploymentControllerName
	job.Labels[constants.LabelOptimizationTarget] = string(instance.Spec.Optimization.Target)

	// Set annotations
	job.Annotations[constants.AnnotationSourceModelUri] = instance.Spec.SourceModel.Uri

	if err := ctrl.SetControllerReference(instance, job, r.Scheme); err != nil {
		return nil, err
	}

	return job, nil
}

func (r *ModelDeploymentReconciler) buildDesiredComponents(ctx context.Context, instance *n8sv1alpha1.ModelDeployment, logger logr.Logger) (*components, error) {
	result := &components{}
	job, err := r.buildOptimizationJob(instance)
	if err != nil {
		logger.Error(err, "unable to construct optimization job")
		return result, err
	}
	result.optimizationJob = job
	return result, nil
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

// reconcileOptimizationJob Reconcile the model optimization job
func (r *ModelDeploymentReconciler) reconcileOptimizationJob(ctx context.Context, instance *n8sv1alpha1.ModelDeployment, c *components, logger logr.Logger) (ctrl.Result, error) {
	// Fetch optimization job
	var job = new(batchv1.Job)
	jobNamespacedName := types.NamespacedName{
		Namespace: c.optimizationJob.Namespace,
		Name:      c.optimizationJob.Name,
	}
	err := r.Get(ctx, jobNamespacedName, job)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to fetch optimization job")
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, constants.EventInternalError, err.Error())
		instance.Status.State = n8sv1alpha1.StatusStateFailed
		return ctrl.Result{}, err
	}

	// If job does not exist then create it
	if apierrors.IsNotFound(err) == true {
		if err := r.Client.Create(ctx, c.optimizationJob); err != nil {
			logger.Error(err, "unable to create optimization job", "Job", c.optimizationJob)
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, constants.EventInternalError, err.Error())
			instance.Status.State = n8sv1alpha1.StatusStateFailed
			return ctrl.Result{}, err
		}
		logger.Info("created new optimization job", "Job", c.optimizationJob)
		r.EventRecorder.Event(
			instance,
			corev1.EventTypeNormal,
			"ModelOptimizationStarted",
			"Started model optimization job",
		)
		instance.Status.State = n8sv1alpha1.StatusStateDeployingModel
		return ctrl.Result{}, nil
	}

	// If current source model URI != URI in spec then delete the job so that it gets re-created with the right URI
	if val, ok := job.Annotations[constants.AnnotationSourceModelUri]; ok {
		// skip if the job is already being deleted
		if !job.DeletionTimestamp.IsZero() {
			return ctrl.Result{}, nil
		}
		if val != instance.Spec.SourceModel.Uri {
			logger.Info("source model URI changed, recreating optimization job")
			r.EventRecorder.Event(instance, corev1.EventTypeNormal, constants.EventModelDeploymentUpdated, "Source model URI updated")
			if err := deleteJob(ctx, r.Client, job); err != nil {
				logger.Error(err, "unable to delete optimization job", "Job", c.optimizationJob)
				r.EventRecorder.Event(instance, corev1.EventTypeWarning, constants.EventInternalError, err.Error())
				instance.Status.State = n8sv1alpha1.StatusStateFailed
				return ctrl.Result{}, err
			}
			logger.Info("optimization job deleted", "Job", job.Name)
			instance.Status.State = n8sv1alpha1.StatusStateOptimizingModel
			return ctrl.Result{}, nil
		}
	}

	// If current optimization target != target in spec then delete the job so that it gets re-created
	if val, ok := job.Labels[constants.LabelOptimizationTarget]; ok {
		// skip if the job is already being deleted
		if !job.DeletionTimestamp.IsZero() {
			return ctrl.Result{}, nil
		}
		if val != string(instance.Spec.Optimization.Target) {
			logger.Info("optimization target changed, recreating optimization job")
			r.EventRecorder.Event(instance, corev1.EventTypeNormal, constants.EventModelDeploymentUpdated, "Optimization target updated")
			if err := deleteJob(ctx, r.Client, job); err != nil {
				logger.Error(err, "unable to delete optimization job", "Job", c.optimizationJob)
				r.EventRecorder.Event(instance, corev1.EventTypeWarning, constants.EventInternalError, err.Error())
				instance.Status.State = n8sv1alpha1.StatusStateFailed
				return ctrl.Result{}, err
			}
			logger.Info("optimization job deleted", "Job", job.Name)
			instance.Status.State = n8sv1alpha1.StatusStateOptimizingModel
			return ctrl.Result{}, nil
		}
	}

	// Check if job finished
	finished, status := isJobFinished(job)

	// If the job failed just record the event and do nothing
	if finished == true && status == batchv1.JobFailed {
		errMsg := job.Status.Conditions[len(job.Status.Conditions)-1].Message
		logger.Error(fmt.Errorf(errMsg), "optimization job failed")
		r.EventRecorder.Eventf(
			instance,
			corev1.EventTypeWarning,
			constants.EventModelOptimizationFailed,
			"Error optimizing model, for more info: kubectl logs job/%s",
			job.Name,
		)
		instance.Status.State = n8sv1alpha1.StatusStateFailed
		return ctrl.Result{}, nil
	}

	// If the job finished successfully and the optimized model is available then deploy the optimized model
	if finished == true && status == batchv1.JobComplete {
		instance.Status.State = n8sv1alpha1.StatusStateDeployingModel
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *ModelDeploymentReconciler) reconcileModelDeployment(ctx context.Context, instance *n8sv1alpha1.ModelDeployment, c *components, logger logr.Logger) (n8sv1alpha1.StatusState, error) {
	var state = n8sv1alpha1.StatusStateDeployingModel

	return state, nil
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

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

	// Build desired components
	desiredComponents, err := r.buildDesiredComponents(ctx, instance, logger)
	if err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, constants.EventInternalError, err.Error())
		instance.Status.State = n8sv1alpha1.StatusStateFailed
		r.updateStatus(ctx, instance, logger)
		return ctrl.Result{}, err
	}

	// Reconcile optimization job
	res, err = r.reconcileOptimizationJob(ctx, instance, desiredComponents, logger)
	if err != nil {
		instance.Status.State = n8sv1alpha1.StatusStateFailed
		r.updateStatus(ctx, instance, logger)
		return ctrl.Result{}, err
	}

	// If optimization job completed successfully then reconcile model deployment
	//if state == n8sv1alpha1.StatusStateDeployingModel {
	//	state, err = r.reconcileModelDeployment(ctx, instance, desiredComponents, logger)
	//	if err != nil {
	//		instance.Status.State = n8sv1alpha1.StatusStateFailed
	//		r.updateStatus(ctx, instance, logger)
	//		return ctrl.Result{}, err
	//	}
	//}

	r.updateStatus(ctx, instance, logger)
	return res, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelDeploymentReconciler) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&n8sv1alpha1.ModelDeployment{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
