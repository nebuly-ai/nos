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

package core

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/nebuly-ai/nebulnetes/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type Controller struct {
	client.Client
	Scheme            *runtime.Scheme
	podBatcher        util.Batcher[v1.Pod]
	clusterState      *state.ClusterState
	currentBatch      map[string]v1.Pod
	planner           Planner
	actuator          Actuator
	lastAppliedPlanId string
}

func NewController(
	scheme *runtime.Scheme,
	client client.Client,
	podBatcher util.Batcher[v1.Pod],
	clusterState *state.ClusterState,
	planner Planner,
	actuator Actuator) Controller {
	return Controller{
		Scheme:       scheme,
		Client:       client,
		clusterState: clusterState,
		currentBatch: make(map[string]v1.Pod),
		podBatcher:   podBatcher,
		planner:      planner,
		actuator:     actuator,
	}
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=persistentvolumes;persistentvolumeclaims;namespaces;services;replicationcontrollers,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=statefulsets;replicasets,verbs=get;list;watch
//+kubebuilder:rbac:groups=storage.k8s.io,resources=csinodes;storageclasses;csidrivers;csistoragecapacities,verbs=get;list;watch
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=elasticquotas,verbs=get;list;watch;
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=compositeelasticquotas,verbs=get;list;watch

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(3).Info("*** start reconcile ***")
	defer logger.V(3).Info("*** end reconcile ***")

	// Fetch instance
	var instance v1.Pod
	if err := c.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if last plan has been reported
	if c.lastAppliedPlanId != "" {
		if reported := c.lastPlanReportedByAllNodes(); !reported {
			logger.Info("last partitioning plan has not been reported by all nodes yet, waiting...")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	// Add pod to current batch only if it is pending and adding extra resources could make it schedulable
	if !pod.ExtraResourcesCouldHelpScheduling(instance) {
		logger.V(3).Info("pod does not require extra resources to be scheduled, skipping it",
			"pod",
			instance.Name,
			"namespace",
			instance.Namespace,
		)
		return ctrl.Result{}, nil
	}

	// Add Pod to current batch only if not already present
	if _, ok := c.currentBatch[util.GetNamespacedName(&instance).String()]; !ok {
		c.podBatcher.Add(instance)
		c.currentBatch[util.GetNamespacedName(&instance).String()] = instance
		logger.V(1).Info("batch updated", "pod", instance.Name, "namespace", instance.Namespace)
	}

	// If batch is ready then process pending pods
	select {
	case <-c.podBatcher.Ready():
		logger.V(1).Info("batch ready")
		c.currentBatch = make(map[string]v1.Pod)
		err := c.processPendingPods(ctx)
		return ctrl.Result{}, err
	default:
		logger.V(1).Info("batch not ready")
	}

	// If batch is not ready then requeue after 1 second
	if len(c.currentBatch) > 0 {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	// If batch is empty and there are no new pending pods, requeue after some time
	// to try to process again the pending pods
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (c *Controller) processPendingPods(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("processing pending pods")

	// Fetch pending pods
	allPendingPods, err := c.fetchPendingPods(ctx)
	if err != nil {
		logger.Error(err, "unable to fetch pending pods")
		return err
	}
	logger.Info(fmt.Sprintf("found %d pending pods", len(allPendingPods)))
	if len(allPendingPods) == 0 {
		return nil
	}

	// Extract pods that can be helped with extra resources
	var pods = make([]v1.Pod, 0)
	for _, p := range allPendingPods {
		if pod.ExtraResourcesCouldHelpScheduling(p) {
			pods = append(pods, p)
		}
	}
	logger.Info(fmt.Sprintf("%d out of %d pending pods could be helped", len(allPendingPods), len(pods)))
	if len(allPendingPods) == 0 {
		return nil
	}

	snapshot := c.clusterState.GetSnapshot()

	// Compute desired state
	plan, err := c.planner.Plan(ctx, snapshot, pods)
	if err != nil {
		logger.Error(err, "unable to plan desired partitioning state")
		return err
	}
	logger.Info("computed desired partitioning state", "partitioning", plan)

	// Apply partitioning plan
	applied, err := c.actuator.Apply(ctx, snapshot, plan)
	if err != nil {
		logger.Error(err, "unable to apply desired partitioning state")
		return err
	}

	if applied {
		c.lastAppliedPlanId = plan.GetId()
	}

	return nil
}

func (c *Controller) fetchPendingPods(ctx context.Context) ([]v1.Pod, error) {
	var podList v1.PodList
	if err := c.List(ctx, &podList, client.MatchingFields{constant.PodPhaseKey: string(v1.PodPending)}); err != nil {
		return nil, err
	}
	return util.Filter(podList.Items, func(pod v1.Pod) bool {
		return pod.Spec.NodeName == ""
	}), nil
}

func (c *Controller) lastPlanReportedByAllNodes() bool {
	nodes := c.clusterState.GetNodes()
	for _, n := range nodes {
		if !c.hasReportedLastPlan(*n.Node()) {
			return false
		}
	}
	return true
}

func (c *Controller) hasReportedLastPlan(n v1.Node) bool {
	val, ok := n.Annotations[v1alpha1.AnnotationReportedPartitioningPlan]
	if !ok {
		return false
	}
	return val == c.lastAppliedPlanId
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Named(name).
		Complete(c)
}
