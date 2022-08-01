package components

import (
	"context"
	"fmt"
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/constants"
	"github.com/nebuly-ai/nebulnetes/utils"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ModelDeploymentComponentLoader struct {
	client               client.Client
	instance             *n8sv1alpha1.ModelDeployment
	optimizationJobCache map[string]struct {
		checkResult constants.ExistenceCheckResult
		job         *batchv1.Job
	}
	analysisJobCache map[string]struct {
		checkResult constants.ExistenceCheckResult
		job         *batchv1.Job
	}
}

func NewModelDeploymentComponentLoader(client client.Client, instance *n8sv1alpha1.ModelDeployment) *ModelDeploymentComponentLoader {
	return &ModelDeploymentComponentLoader{
		client:   client,
		instance: instance,
		optimizationJobCache: make(map[string]struct {
			checkResult constants.ExistenceCheckResult
			job         *batchv1.Job
		}),
		analysisJobCache: make(map[string]struct {
			checkResult constants.ExistenceCheckResult
			job         *batchv1.Job
		}),
	}
}

func (r *ModelDeploymentComponentLoader) CheckOptimizationJobExists(ctx context.Context) (constants.ExistenceCheckResult, *batchv1.Job, error) {
	logger := log.FromContext(ctx)

	namespacedName := utils.GetNamespacedName(r.instance)
	if val, ok := r.optimizationJobCache[namespacedName]; ok {
		logger.V(1).Info("using cached result for checking optimization job existence")
		return val.checkResult, val.job, nil
	}

	var jobList = new(batchv1.JobList)
	listOption := GetOptimizationJobListFilter(r.instance)
	logger.V(1).Info("loading optimization job", "ModelDeployment", namespacedName)
	err := r.client.List(ctx, jobList, listOption)
	if err != nil {
		return constants.ExistenceCheckError, nil, errors.Wrap(err, "unable to fetch optimization job")
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

func (r *ModelDeploymentComponentLoader) CheckAnalysisJobExists(ctx context.Context) (constants.ExistenceCheckResult, *batchv1.Job, error) {
	logger := log.FromContext(ctx)

	namespacedName := utils.GetNamespacedName(r.instance)
	if val, ok := r.optimizationJobCache[namespacedName]; ok {
		logger.V(1).Info("using cached result for checking analysis job existence")
		return val.checkResult, val.job, nil
	}

	var jobList = new(batchv1.JobList)
	listOption := GetAnalysisJobListFilter(r.instance)
	logger.V(1).Info("loading analysis job", "ModelDeployment", namespacedName)
	err := r.client.List(ctx, jobList, listOption)
	if err != nil {
		return constants.ExistenceCheckError, nil, errors.Wrap(err, "unable to fetch analysis job")
	}
	if len(jobList.Items) == 0 {
		return constants.ExistenceCheckCreate, nil, nil
	}
	if len(jobList.Items) == 1 {
		return constants.ExistenceCheckExists, &jobList.Items[0], nil
	}
	err = fmt.Errorf(
		"model deployments should have only one analysis job, but %d were found",
		len(jobList.Items),
	)
	return constants.ExistenceCheckError, nil, err
}

func GetOptimizationJobListFilter(m *n8sv1alpha1.ModelDeployment) client.ListOption {
	return client.MatchingLabels{
		constants.LabelCreatedBy:       constants.ModelDeploymentControllerName,
		constants.LabelModelDeployment: m.GetName(),
		constants.LabelJobKind:         constants.JobKindModelOptimization,
	}
}

func GetAnalysisJobListFilter(modelDeployment *n8sv1alpha1.ModelDeployment) client.ListOption {
	return client.MatchingLabels{
		constants.LabelCreatedBy:       constants.ModelDeploymentControllerName,
		constants.LabelModelDeployment: modelDeployment.GetName(),
		constants.LabelJobKind:         constants.JobKindModelAnalysis,
	}
}
