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
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"github.com/nebuly-ai/nebulnetes/pkg/util/pod"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sort"
	"time"
)

type Controller struct {
	client.Client
	Scheme       *runtime.Scheme
	podBatcher   util.Batcher[v1.Pod]
	clusterState *state.ClusterState
	currentBatch map[string]v1.Pod
	planner      Planner
	actuator     Actuator
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
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(3).Info("*** start reconcile ***")
	defer logger.V(3).Info("*** end reconcile ***")

	// Fetch instance
	var instance v1.Pod
	if err := c.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
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
		return c.processPendingPods(ctx)
	default:
		logger.V(1).Info("batch not ready")
	}

	if len(c.currentBatch) > 0 {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (c *Controller) processPendingPods(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("processing pending pods")

	// Fetch pending pods
	allPendingPods, err := c.fetchPendingPods(ctx)
	if err != nil {
		logger.Error(err, "unable to fetch pending pods")
		return ctrl.Result{}, err
	}
	logger.Info(fmt.Sprintf("found %d pending pods", len(allPendingPods)))
	if len(allPendingPods) == 0 {
		return ctrl.Result{}, nil
	}

	// Extract pods that can be helped with extra resources
	var pods = make([]v1.Pod, 0)
	for _, p := range allPendingPods {
		if pod.ExtraResourcesCouldHelpScheduling(p) {
			pods = append(pods, p)
		}
	}
	logger.Info(fmt.Sprintf("%d out of %d pending pods can be helped", len(allPendingPods), len(pods)))
	if len(allPendingPods) == 0 {
		return ctrl.Result{}, nil
	}

	snapshot := c.clusterState.GetSnapshot()

	// Sort Pods by importance
	sort.Slice(pods, func(i, j int) bool {
		return pod.IsMoreImportant(pods[i], pods[j])
	})

	// Compute desired state
	desiredState, err := c.planner.Plan(ctx, snapshot, pods)
	if err != nil {
		logger.Error(err, "unable to plan desired partitioning state")
		return ctrl.Result{}, err
	}
	logger.Info("computed desired partitioning state", "partitioning", desiredState)

	// Apply partitioning plan
	if err = c.actuator.Apply(ctx, snapshot, desiredState); err != nil {
		logger.Error(err, "unable to apply desired partitioning state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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

func (c *Controller) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Named(name).
		Complete(c)
}
