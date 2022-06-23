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
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	Scheme *runtime.Scheme
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
		},
	}
}

func (r *ModelDeploymentReconciler) constructOptimizationJob(modelDeployment *n8sv1alpha1.ModelDeployment) (*batchv1.Job, error) {
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

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=modeldeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

func (r *ModelDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var modelDeployment n8sv1alpha1.ModelDeployment
	if err := r.Get(ctx, req.NamespacedName, &modelDeployment); err != nil {
		logger.Error(err, "unable to fetch ModelDeployment")
		return ctrl.Result{}, err
	}

	// Fetch optimization job
	var optimizationJob batchv1.Job
	jobNamespacedName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      fmt.Sprintf(optimizationJobNameFormat, req.Name),
	}
	err := r.Get(ctx, jobNamespacedName, &optimizationJob)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to fetch optimization job")
		return ctrl.Result{}, err
	}

	// If job does not exist then create it
	if apierrors.IsNotFound(err) {
		newJob, err := r.constructOptimizationJob(&modelDeployment)
		if err != nil {
			logger.Error(err, "unable to construct optimization job")
			// don't reschedule until we get a change to the spec
			return ctrl.Result{}, nil
		}
		if err := r.Client.Create(ctx, newJob); err != nil {
			logger.Error(err, "unable to create optimization job", "job", newJob)
			return ctrl.Result{}, err
		}
		logger.Info("created new optimization job", "job", newJob)
		return ctrl.Result{}, nil
	}

	finished, status := isJobFinished(&optimizationJob)
	if finished == true && status == batchv1.JobFailed {
		errMsg := optimizationJob.Status.Conditions[len(optimizationJob.Status.Conditions)-1].Message
		logger.Error(fmt.Errorf(errMsg), "unable to perform model optimization")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&n8sv1alpha1.ModelDeployment{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
