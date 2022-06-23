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
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
)

const (
	// Format used for generating the names of the optimization jobs
	optimizationJobNameFormat = "%s-optimization"
	// Number of retries before declaring an optimization job failed
	optimizationJobBackoffLimit int32 = 0
	// Name of the Docker image used for optimizing models for inference
	modelOptimizerImageName = "nebuly.ai/model-optimizer"
)

// ModelDeploymentReconciler reconciles a ModelDeployment object
type ModelDeploymentReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

type desiredComponents struct {
	optimizationJob *batchv1.Job
}

func isJobFinished(job *batchv1.Job) (bool, batchv1.JobConditionType) {
	for _, c := range job.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == v1.ConditionTrue {
			return true, c.Type
		}
	}

	return false, ""
}

func constructInferenceOptimizationService(modelDeployment *n8sv1alpha1.ModelDeployment) *v1.Container {
	modelOptimizerImage := fmt.Sprintf(
		"%s:%s",
		modelOptimizerImageName,
		modelDeployment.Spec.Optimization.ModelOptimizerVersion,
	)
	return &v1.Container{
		Name:                     "optimizer",
		Image:                    modelOptimizerImage,
		TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
		Env: []v1.EnvVar{
			{
				Name:  "MODEL_DEPLOYMENT_NAME",
				Value: modelDeployment.Name,
			},
		},
		Args: []string{
			modelDeployment.Spec.ModelUri,
			modelDeployment.Spec.ModelLibraryUri,
			string(modelDeployment.Spec.Optimization.Target),
		},
	}
}

func (r *ModelDeploymentReconciler) buildOptimizationJob(modelDeployment *n8sv1alpha1.ModelDeployment) (*batchv1.Job, error) {
	optimizationJobBackoffLimitVar := optimizationJobBackoffLimit
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        fmt.Sprintf(optimizationJobNameFormat, modelDeployment.Name),
			Namespace:   modelDeployment.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &optimizationJobBackoffLimitVar,
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers:         []v1.Container{*constructInferenceOptimizationService(modelDeployment)},
					ServiceAccountName: "default", // todo
					RestartPolicy:      v1.RestartPolicyNever,
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(modelDeployment, job, r.Scheme); err != nil {
		return nil, err
	}
	return job, nil
}

func (r *ModelDeploymentReconciler) buildDesiredComponents(ctx context.Context, modelDeployment n8sv1alpha1.ModelDeployment, logger logr.Logger) (*desiredComponents, error) {
	result := &desiredComponents{}
	job, err := r.buildOptimizationJob(&modelDeployment)
	if err != nil {
		logger.Error(err, "unable to construct optimization job")
		return result, err
	}
	result.optimizationJob = job
	return result, nil
}

func (r *ModelDeploymentReconciler) updateStatus(ctx context.Context, desiredModelDeployment n8sv1alpha1.ModelDeployment, logger logr.Logger) {
	var currentModelDeployment n8sv1alpha1.ModelDeployment
	namespacedName := types.NamespacedName{Name: desiredModelDeployment.Name, Namespace: desiredModelDeployment.Namespace}
	if err := r.Get(ctx, namespacedName, &currentModelDeployment); err != nil {
		logger.Error(err, "unable to fetch ModelDeployment")
		return
	}
	if equality.Semantic.DeepEqual(currentModelDeployment.Status, desiredModelDeployment.Status) {
		logger.V(1).Info("current status and desired status of ModelDeployment are equal, skipping update")
		return
	}
	logger.Info("Updating ModelDeployment status", "ModelDeployment", desiredModelDeployment.Name)
	if err := r.Status().Update(ctx, &desiredModelDeployment); err != nil {
		logger.Error(err, "unable to update ModelDeployment status")
		r.EventRecorder.Eventf(
			&desiredModelDeployment,
			v1.EventTypeWarning,
			"StatusUpdateFailed",
			"Failed to update status of ModelDeployment %q: %s", desiredModelDeployment.Name, err.Error(),
		)
	}
}

// reconcileOptimizationJob Reconcile the model optimization job
func (r *ModelDeploymentReconciler) reconcileOptimizationJob(ctx context.Context, modelDeployment n8sv1alpha1.ModelDeployment, c *desiredComponents, logger logr.Logger) (n8sv1alpha1.StatusState, error) {
	var state = n8sv1alpha1.StatusStateOptimizingModel

	// Fetch optimization job
	var optimizationJob batchv1.Job
	jobNamespacedName := types.NamespacedName{
		Namespace: c.optimizationJob.Namespace,
		Name:      c.optimizationJob.Name,
	}
	err := r.Get(ctx, jobNamespacedName, &optimizationJob)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to fetch optimization job")
		r.EventRecorder.Event(&modelDeployment, v1.EventTypeWarning, EventInternalError, err.Error())
		return n8sv1alpha1.StatusStateFailed, err
	}

	// If job does not exist then create it
	if apierrors.IsNotFound(err) == true {
		if err := r.Client.Create(ctx, c.optimizationJob); err != nil {
			logger.Error(err, "unable to create optimization job", "Job", c.optimizationJob)
			r.EventRecorder.Event(&modelDeployment, v1.EventTypeWarning, EventInternalError, err.Error())
			return n8sv1alpha1.StatusStateFailed, err
		}
		logger.Info("created new optimization job", "Job", c.optimizationJob)
		r.EventRecorder.Event(
			&modelDeployment,
			v1.EventTypeNormal,
			"ModelOptimizationStarted",
			"Started job for optimizing the deployed model",
		)
	} else {
		// Update the referenced job object in the status
		ref, err := reference.GetReference(r.Scheme, &optimizationJob)
		if err != nil {
			logger.Error(
				err,
				"unable to make reference to model optimization job",
				"job",
				optimizationJob,
			)
			r.EventRecorder.Event(&modelDeployment, v1.EventTypeWarning, EventInternalError, err.Error())
			return n8sv1alpha1.StatusStateFailed, err
		}
		modelDeployment.Status.ModelOptimizationJob = *ref

		// Check if job finished
		finished, status := isJobFinished(&optimizationJob)

		// If the job failed just record the event and do nothing
		if finished == true && status == batchv1.JobFailed {
			state = n8sv1alpha1.StatusStateFailed
			errMsg := optimizationJob.Status.Conditions[len(optimizationJob.Status.Conditions)-1].Message
			logger.Error(fmt.Errorf(errMsg), "unable to perform model optimization")
			r.EventRecorder.Eventf(
				&modelDeployment,
				v1.EventTypeWarning,
				EventModelOptimizationFailed,
				"Error optimizing model, for more information run: kubectl logs job/%s",
				optimizationJob.Name,
			)
		}
		// If the job finished successfully then deploy the optimized model
		if finished == true && status == batchv1.JobComplete {
			state = n8sv1alpha1.StatusStateDeployingModel
			r.EventRecorder.Event(
				&modelDeployment,
				v1.EventTypeWarning,
				EventModelOptimizationCompleted,
				"Model optimized successfully",
			)
		}
	}
	return state, nil
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

func (r *ModelDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch ModelDeployment
	var modelDeployment n8sv1alpha1.ModelDeployment
	if err := r.Get(ctx, req.NamespacedName, &modelDeployment); err != nil {
		logger.Error(err, "unable to fetch ModelDeployment")
		return ctrl.Result{}, err
	}

	// Build desired components
	desiredComponents, err := r.buildDesiredComponents(ctx, modelDeployment, logger)
	if err != nil {
		r.EventRecorder.Event(&modelDeployment, v1.EventTypeWarning, EventInternalError, err.Error())
		modelDeployment.Status.State = n8sv1alpha1.StatusStateFailed
		r.updateStatus(ctx, modelDeployment, logger)
		return ctrl.Result{}, err
	}

	// Reconcile optimization job
	state, err := r.reconcileOptimizationJob(ctx, modelDeployment, desiredComponents, logger)
	if err != nil {
		modelDeployment.Status.State = n8sv1alpha1.StatusStateFailed
		r.updateStatus(ctx, modelDeployment, logger)
		return ctrl.Result{}, err
	}
	modelDeployment.Status.State = state

	r.updateStatus(ctx, modelDeployment, logger)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&n8sv1alpha1.ModelDeployment{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
