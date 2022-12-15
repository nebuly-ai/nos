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
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/nebuly-ai/nebulnetes/pkg/util/predicate"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

type MigReporter struct {
	client.Client
	migClient       mig.Client
	refreshInterval time.Duration
	sharedState     *SharedState
}

func NewReporter(client client.Client, migClient mig.Client, sharedState *SharedState, refreshInterval time.Duration) MigReporter {
	reporter := MigReporter{
		Client:          client,
		migClient:       migClient,
		sharedState:     sharedState,
		refreshInterval: refreshInterval,
	}
	return reporter
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch

func (r *MigReporter) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := klog.FromContext(ctx).WithName("Reporter")

	r.sharedState.Lock()
	defer r.sharedState.Unlock()
	defer r.sharedState.OnReportDone()

	var instance v1.Node
	if err := r.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}

	// Compute new status annotations
	migResources, err := r.migClient.GetMigDevices(ctx)
	if err != nil {
		logger.Error(err, "unable to get MIG device resources")
		return ctrl.Result{}, err
	}
	usedMigs := migResources.GetUsed()
	freeMigs := migResources.GetFree()
	logger.V(3).Info("loaded free MIG devices", "freeMIGs", freeMigs)
	logger.V(3).Info("loaded used MIG devices", "usedMIGs", usedMigs)
	newStatusAnnotations := mig.ComputeStatusAnnotations(migResources)

	// Get current status annotations and compare with new ones
	oldStatusAnnotations, _ := mig.GetGPUAnnotationsFromNode(instance)
	if util.UnorderedEqual(newStatusAnnotations, oldStatusAnnotations) {
		if instance.Annotations[v1alpha1.AnnotationReportedPartitioningPlan] == r.sharedState.lastParsedPlanId {
			logger.Info("current status is equal to last reported status, nothing to do")
			return ctrl.Result{RequeueAfter: r.refreshInterval}, nil
		}
	}

	// Update node
	logger.Info("status changed - reporting it by updating node annotations")
	updated := instance.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = make(map[string]string)
	}
	for k := range updated.Annotations {
		if strings.HasPrefix(k, v1alpha1.AnnotationGPUStatusPrefix) {
			delete(updated.Annotations, k)
		}
	}
	for _, a := range newStatusAnnotations {
		updated.Annotations[a.String()] = a.GetValue()
	}
	updated.Annotations[v1alpha1.AnnotationReportedPartitioningPlan] = r.sharedState.lastParsedPlanId
	if err := r.Client.Patch(ctx, updated, client.MergeFrom(&instance)); err != nil {
		logger.Error(err, "unable to update node status annotations", "annotations", updated.Annotations)
		return ctrl.Result{}, err
	}

	logger.Info("updated reported status - node annotations updated successfully")

	return ctrl.Result{RequeueAfter: r.refreshInterval}, nil
}

func (r *MigReporter) SetupWithManager(mgr ctrl.Manager, controllerName string, nodeName string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&v1.Node{},
			builder.WithPredicates(
				predicate.ExcludeDelete{},
				predicate.MatchingName{Name: nodeName},
				predicate.NodeResourcesChanged{},
			),
		).
		Named(controllerName).
		Complete(r)
}
