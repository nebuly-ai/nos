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
			Labels:       make(map[string]string),
			Annotations:  make(map[string]string),
			GenerateName: constants.OptimizationJobNamePrefix,
			Namespace:    instance.Namespace,
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
	job.Labels[constants.LabelModelDeployment] = instance.GetName()

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

	var jobList = new(batchv1.JobList)
	err := r.GetClient().List(ctx, jobList, client.MatchingLabels{constants.LabelModelDeployment: r.instance.GetName()})
	if err != nil {
		logger.Error(err, "unable to fetch optimization job")
		return constants.ExistenceCheckError, nil, err
	}
	if len(jobList.Items) == 0 {
		return constants.ExistenceCheckCreate, nil, nil
	}
	if len(jobList.Items) == 1 {
		return constants.ExistenceCheckExists, &jobList.Items[0], nil
	}
	err = fmt.Errorf(
		"model deployments should have only one optimization job, but %d were found",
		len(jobList.Items),
	)
	return constants.ExistenceCheckError, nil, err
}

func (r *OptimizationJobReconciler) checkModelDescriptorConfigMapExists(ctx context.Context) (constants.ExistenceCheckResult, *corev1.ConfigMap, error) {
	logger := log.FromContext(ctx)

	var configMapList = new(corev1.ConfigMapList)
	labels := client.MatchingLabels{}
	for k, v := range GetModelDescriptorConfigMapLabels(r.instance, r.optimizationJob) {
		labels[k] = v
	}

	err := r.GetClient().List(ctx, configMapList, labels)
	if err != nil {
		logger.Error(err, "unable to fetch optimization job")
		return constants.ExistenceCheckError, nil, err
	}
	if len(configMapList.Items) == 0 {
		return constants.ExistenceCheckCreate, nil, nil
	}
	if len(configMapList.Items) == 1 {
		return constants.ExistenceCheckExists, &configMapList.Items[0], nil
	}
	err = fmt.Errorf(
		"model optimization jobs should have only one model descriptor config map, but %d were found",
		len(configMapList.Items),
	)
	return constants.ExistenceCheckError, nil, err
}

func (r *OptimizationJobReconciler) buildModelDescriptorConfigMap(modelDescriptor *components.ModelDescriptor) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    r.instance.Namespace,
			GenerateName: constants.ModelDescriptorNamePrefix,
			Labels:       GetModelDescriptorConfigMapLabels(r.instance, r.optimizationJob),
		},
		Immutable: utils.BoolAddr(true),
		Data:      modelDescriptor.AsMap(),
	}
}

func (r *OptimizationJobReconciler) createOptimizationJob(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
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

func (r *OptimizationJobReconciler) createModelDescriptorConfigMap(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	modelDescriptor, err := r.modelLibrary.FetchOptimizedModelDescriptor(ctx, r.instance)
	if err != nil {
		logger.Error(err, "unable to fetch model descriptor", "uri", r.modelLibrary.GetOptimizedModelDescriptorUri(r.instance))
		return ctrl.Result{}, err
	}

	modelDescriptorConfigMap := r.buildModelDescriptorConfigMap(modelDescriptor)
	if err = r.CreateResourceIfNotExists(ctx, r.instance, modelDescriptorConfigMap); err != nil {
		logger.Error(
			err,
			"unable to create optimized model descriptor configmap",
			"ConfigMap",
			modelDescriptorConfigMap,
		)
		return ctrl.Result{}, err
	}
	logger.Info("created model descriptor configmap", "ConfigMap", modelDescriptorConfigMap)
	r.GetRecorder().Event(
		r.instance,
		corev1.EventTypeNormal,
		"ModelOptimizationCompleted",
		"Created model descriptor ConfigMap",
	)

	return ctrl.Result{}, nil
}

func (r *OptimizationJobReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	jobCheckResult, job, err := r.checkOptimizationJobExists(ctx)
	if err != nil {
		return r.HandleError(r.instance, err)
	}
	// If job does not exist then create it
	if jobCheckResult == constants.ExistenceCheckCreate {
		return r.createOptimizationJob(ctx)
	}
	// If the job exists, then get the auto-generated name and update instance status
	if jobCheckResult == constants.ExistenceCheckExists {
		r.optimizationJob.Name = job.Name
		r.instance.Status.OptimizationJob = job.Name
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

	// If the job completed then create the optimized model descriptor configmap, if not already created
	if finished == true && status == batchv1.JobComplete {
		cmCheckResult, modelDescriptorCm, err := r.checkModelDescriptorConfigMapExists(ctx)
		if err != nil {
			return r.HandleError(r.instance, err)
		}
		if cmCheckResult == constants.ExistenceCheckCreate {
			return r.createModelDescriptorConfigMap(ctx)
		}
		// If cm exists, then update the instance status
		if cmCheckResult == constants.ExistenceCheckExists {
			r.instance.Status.OptimizedModelDescriptor = modelDescriptorCm.Name
		}
	}

	return ctrl.Result{}, nil
}

func GetModelDescriptorConfigMapLabels(m *n8sv1alpha1.ModelDeployment, optimizationJob *batchv1.Job) map[string]string {
	return map[string]string{
		constants.LabelModelDeployment: m.Name,
		constants.LabelOptimizationJob: optimizationJob.Name,
	}
}
