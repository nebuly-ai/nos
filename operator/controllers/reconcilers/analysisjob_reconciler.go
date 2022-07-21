package reconcilers

import (
	"context"
	"fmt"
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/constants"
	"github.com/nebuly-ai/nebulnetes/controllers/components"
	"github.com/nebuly-ai/nebulnetes/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type AnalysisJobReconciler struct {
	components.ComponentReconcilerBase
	modelLibrary components.ModelLibrary
	analysisJob  *batchv1.Job
	loader       *components.ModelDeploymentComponentLoader
	instance     *n8sv1alpha1.ModelDeployment
}

func NewAnalysisJobReconciler(client client.Client,
	scheme *runtime.Scheme,
	eventRecorder record.EventRecorder,
	modelLibrary components.ModelLibrary,
	loader *components.ModelDeploymentComponentLoader,
	instance *n8sv1alpha1.ModelDeployment) (*AnalysisJobReconciler, error) {

	job, err := buildAnalysisJob(modelLibrary, instance)
	if err != nil {
		return nil, err
	}
	return &AnalysisJobReconciler{
		ComponentReconcilerBase: components.NewComponentReconcilerBase(client, scheme, eventRecorder),
		modelLibrary:            modelLibrary,
		analysisJob:             job,
		loader:                  loader,
		instance:                instance,
	}, nil
}

func buildAnalysisJob(ml components.ModelLibrary, instance *n8sv1alpha1.ModelDeployment) (*batchv1.Job, error) {
	container, err := buildModelAnalyzerContainer(ml, instance)
	if err != nil {
		return nil, err
	}

	analysisJobBackoffLimit := int32(instance.Spec.Optimization.AnalysisJobBackoffLimit)
	var runAsNonRoot = true

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:       make(map[string]string),
			Annotations:  make(map[string]string),
			GenerateName: constants.AnalysisJobNamePrefix,
			Namespace:    instance.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &analysisJobBackoffLimit,
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
	job.Labels[constants.LabelModelDeployment] = instance.GetName()
	job.Labels[constants.LabelJobKind] = constants.JobKindModelAnalysis

	// Set annotations
	job.Annotations[constants.AnnotationSourceModelUri] = instance.Spec.SourceModel.Uri

	return job, nil
}

func buildModelAnalyzerContainer(ml components.ModelLibrary, md *n8sv1alpha1.ModelDeployment) (*corev1.Container, error) {
	mlCredentials, err := ml.GetCredentials()
	if err != nil {
		return nil, err
	}

	envVars := make([]corev1.EnvVar, 0)
	for key, value := range mlCredentials {
		envVars = append(envVars, corev1.EnvVar{Name: key, Value: value})
	}
	modelAnalyzerImage := fmt.Sprintf(
		"%s:%s",
		md.Spec.Optimization.ModelAnalyzerImageName,
		md.Spec.Optimization.ModelAnalyzerImageVersion,
	)

	return &corev1.Container{
		Name:                     "analyzer",
		Image:                    modelAnalyzerImage,
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		Env:                      envVars,
		Args: []string{
			md.Spec.SourceModel.Uri,
			ml.GetBaseUri(md),
			ml.GetModelDescriptorUri(md),
			string(ml.GetStorageKind()),
			string(md.Spec.Optimization.Target),
		},
	}, nil
}

func (r *AnalysisJobReconciler) createAnalysisJob(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if err := r.CreateResourceIfNotExists(ctx, r.instance, r.analysisJob); err != nil {
		logger.Error(err, "unable to create analysis job", "Job", r.analysisJob)
		return r.HandleError(r.instance, err)
	}
	logger.Info("created new analysis job", "Job", r.analysisJob)
	r.GetRecorder().Event(
		r.instance,
		corev1.EventTypeNormal,
		"ModelOptimizationStarted",
		"Started model optimization job",
	)
	r.instance.Status.State = n8sv1alpha1.StatusStateAnalyzingModel
	return ctrl.Result{}, nil
}

func (r *AnalysisJobReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	jobCheckResult, job, err := r.loader.CheckAnalysisJobExists(ctx)
	if err != nil {
		return r.HandleError(r.instance, err)
	}
	// If job does not exist then create it
	if jobCheckResult == constants.ExistenceCheckCreate {
		return r.createAnalysisJob(ctx)
	}
	// If the job exists, then update the reconciler field and the instance status
	if jobCheckResult == constants.ExistenceCheckExists {
		r.analysisJob = job
		r.instance.Status.AnalysisJob = job.Name
	}

	// If current source model URI != URI in spec then delete the job so that it gets re-created with the right URI
	if val, ok := job.Annotations[constants.AnnotationSourceModelUri]; ok {
		// skip if the job is already being deleted
		if !job.DeletionTimestamp.IsZero() {
			return ctrl.Result{}, nil
		}
		if val != r.instance.Spec.SourceModel.Uri {
			logger.Info("source model URI changed, recreating analysis job")
			r.GetRecorder().Event(r.instance, corev1.EventTypeNormal, constants.EventModelDeploymentUpdated, "Source model URI updated")
			if err := r.DeleteResourceIfExists(ctx, job); err != nil {
				logger.Error(err, "unable to delete analysis job", "Job", r.analysisJob)
				return r.HandleError(r.instance, err)
			}
			logger.Info("analysis job deleted", "Job", job.Name)
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
			logger.Info("optimization target changed, recreating analysis job")
			r.GetRecorder().Event(r.instance, corev1.EventTypeNormal, constants.EventModelDeploymentUpdated, "Optimization target updated")
			if err := r.DeleteResourceIfExists(ctx, job); err != nil {
				logger.Error(err, "unable to delete analysis job", "Job", r.analysisJob)
				return r.HandleError(r.instance, err)
			}
			logger.Info("analysis job deleted", "Job", job.Name)
			return ctrl.Result{}, nil
		}
	}

	// Check if job finished
	finished, status := utils.IsJobFinished(job)

	// If the job failed just record the event and do nothing
	if finished == true && status == batchv1.JobFailed {
		errMsg := job.Status.Conditions[len(job.Status.Conditions)-1].Message
		logger.Error(fmt.Errorf(errMsg), "analysis job failed")
		r.GetRecorder().Eventf(
			r.instance,
			corev1.EventTypeWarning,
			constants.EventModelOptimizationFailed,
			"Error analyzing model, for more info: kubectl logs job/%s",
			job.Name,
		)
		r.instance.Status.State = n8sv1alpha1.StatusStateFailed
		return ctrl.Result{}, nil
	}

	// If the job completed then go on with the reconciliation chain
	if finished == true && status == batchv1.JobComplete {
		return r.Next(ctx)
	}

	return ctrl.Result{}, nil
}
