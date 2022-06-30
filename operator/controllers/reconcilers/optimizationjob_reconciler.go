package reconcilers

import (
	"context"
	"fmt"
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/constants"
	"github.com/nebuly-ai/nebulnetes/controllers/components"
	"github.com/nebuly-ai/nebulnetes/controllers/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Format used for generating the names of the optimization jobs
	optimizationJobNameFormat = "%s-optimization"
)

type OptimizationJobReconciler struct {
	components.ComponentReconcilerBase
	modelLibrary    components.ModelLibrary
	optimizationJob *batchv1.Job
	instance        *n8sv1alpha1.ModelDeployment
}

func NewOptimizationJobReconciler(client client.Client,
	scheme *runtime.Scheme,
	eventRecorder record.EventRecorder,
	modelLibrary components.ModelLibrary,
	instance *n8sv1alpha1.ModelDeployment) (*OptimizationJobReconciler, error) {

	job, err := buildOptimizationJob(modelLibrary, instance)
	if err != nil {
		return nil, err
	}
	return &OptimizationJobReconciler{
		ComponentReconcilerBase: components.NewComponentReconcilerBase(client, scheme, eventRecorder),
		modelLibrary:            modelLibrary,
		optimizationJob:         job,
		instance:                instance,
	}, nil
}

func buildOptimizationJob(ml components.ModelLibrary, instance *n8sv1alpha1.ModelDeployment) (*batchv1.Job, error) {
	container, err := buildModelOptimizerContainer(ml, instance)
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
	job.Labels[constants.LabelCreatedBy] = constants.ModelDeploymentControllerName
	job.Labels[constants.LabelOptimizationTarget] = string(instance.Spec.Optimization.Target)

	// Set annotations
	job.Annotations[constants.AnnotationSourceModelUri] = instance.Spec.SourceModel.Uri

	return job, nil
}

func buildModelOptimizerContainer(ml components.ModelLibrary, md *n8sv1alpha1.ModelDeployment) (*corev1.Container, error) {
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

func (r *OptimizationJobReconciler) checkOptimizationJobExists(ctx context.Context) (constants.ExistenceCheckResult, *batchv1.Job, error) {
	logger := log.FromContext(ctx)

	var job = new(batchv1.Job)
	jobNamespacedName := types.NamespacedName{
		Namespace: r.optimizationJob.Namespace,
		Name:      r.optimizationJob.Name,
	}
	err := r.GetClient().Get(ctx, jobNamespacedName, job)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to fetch optimization job")
		return constants.ExistenceCheckError, nil, err
	}
	if apierrors.IsNotFound(err) {
		return constants.ExistenceCheckCreate, nil, nil
	}
	return constants.ExistenceCheckExists, job, nil
}

func (r *OptimizationJobReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	checkResult, job, err := r.checkOptimizationJobExists(ctx)
	if err != nil {
		return r.HandleError(r.instance, err)
	}

	// If job does not exist then create it
	if checkResult == constants.ExistenceCheckCreate {
		if err := r.CreateResourceIfNotExists(ctx, r.instance, r.optimizationJob); err != nil {
			logger.Error(err, "unable to create optimization job", "Job", r.optimizationJob)
			return r.HandleError(r.instance, err)
		}
		logger.Info("created new optimization job", "Job", r.optimizationJob)
		r.GetRecorder().Event(
			r.instance,
			corev1.EventTypeNormal,
			"ModelOptimizationStarted",
			"Started model optimization job",
		)
		r.instance.Status.State = n8sv1alpha1.StatusStateDeployingModel
		return ctrl.Result{}, nil
	}

	// If current source model URI != URI in spec then delete the job so that it gets re-created with the right URI
	if val, ok := job.Annotations[constants.AnnotationSourceModelUri]; ok {
		// skip if the job is already being deleted
		if !job.DeletionTimestamp.IsZero() {
			return ctrl.Result{}, nil
		}
		if val != r.instance.Spec.SourceModel.Uri {
			logger.Info("source model URI changed, recreating optimization job")
			r.GetRecorder().Event(r.instance, corev1.EventTypeNormal, constants.EventModelDeploymentUpdated, "Source model URI updated")
			if err := r.DeleteResourceIfExists(ctx, job); err != nil {
				logger.Error(err, "unable to delete optimization job", "Job", r.optimizationJob)
				return r.HandleError(r.instance, err)
			}
			logger.Info("optimization job deleted", "Job", job.Name)
			r.instance.Status.State = n8sv1alpha1.StatusStateOptimizingModel
			return ctrl.Result{}, nil
		}
	}

	// If current optimization target != target in spec then delete the job so that it gets re-created
	if val, ok := job.Labels[constants.LabelOptimizationTarget]; ok {
		// skip if the job is already being deleted
		if !job.DeletionTimestamp.IsZero() {
			return ctrl.Result{}, nil
		}
		if val != string(r.instance.Spec.Optimization.Target) {
			logger.Info("optimization target changed, recreating optimization job")
			r.GetRecorder().Event(r.instance, corev1.EventTypeNormal, constants.EventModelDeploymentUpdated, "Optimization target updated")
			if err := r.DeleteResourceIfExists(ctx, job); err != nil {
				logger.Error(err, "unable to delete optimization job", "Job", r.optimizationJob)
				return r.HandleError(r.instance, err)
			}
			logger.Info("optimization job deleted", "Job", job.Name)
			r.instance.Status.State = n8sv1alpha1.StatusStateOptimizingModel
			return ctrl.Result{}, nil
		}
	}

	// Check if job finished
	finished, status := utils.IsJobFinished(job)

	// If the job failed just record the event and do nothing
	if finished == true && status == batchv1.JobFailed {
		errMsg := job.Status.Conditions[len(job.Status.Conditions)-1].Message
		logger.Error(fmt.Errorf(errMsg), "optimization job failed")
		r.GetRecorder().Eventf(
			r.instance,
			corev1.EventTypeWarning,
			constants.EventModelOptimizationFailed,
			"Error optimizing model, for more info: kubectl logs job/%s",
			job.Name,
		)
		r.instance.Status.State = n8sv1alpha1.StatusStateFailed
		return ctrl.Result{}, nil
	}

	// If the job finished successfully and the optimized model is available then deploy the optimized model
	if finished == true && status == batchv1.JobComplete {
		r.instance.Status.State = n8sv1alpha1.StatusStateDeployingModel
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}
