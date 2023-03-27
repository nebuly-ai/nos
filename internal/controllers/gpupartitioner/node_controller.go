/*
 * Copyright 2023 nebuly.com.
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

package gpupartitioner

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.com/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	"github.com/nebuly-ai/nos/pkg/gpu"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type NodeController struct {
	client.Client
	Scheme         *runtime.Scheme
	clusterState   *state.ClusterState
	migInitializer core.NodeInitializer
}

func NewNodeController(
	client client.Client,
	scheme *runtime.Scheme,
	migInitializer core.NodeInitializer,
	state *state.ClusterState,
) NodeController {
	return NodeController{
		Client:         client,
		Scheme:         scheme,
		clusterState:   state,
		migInitializer: migInitializer,
	}
}

func (c *NodeController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch instance
	var instance v1.Node
	objKey := client.ObjectKey{Namespace: req.Namespace, Name: req.Name}
	err := c.Client.Get(ctx, objKey, &instance)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to fetch node")
		return ctrl.Result{}, err
	}
	if apierrors.IsNotFound(err) {
		logger.V(2).Info("deleting node", "node", instance.Name)
		c.clusterState.DeleteNode(instance.Name)
		return ctrl.Result{}, nil
	}

	// Handle MIG node initialization
	var initialized = true
	if gpu.IsMigPartitioningEnabled(instance) {
		_, specAnnotations := gpu.ParseNodeAnnotations(instance)
		if len(specAnnotations) == 0 {
			initialized = false
			if err = c.migInitializer.InitNodePartitioning(ctx, instance); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to initialize node MIG partitioning: %w", err)
			}
		}
	}

	// If the node is not initialized, do not add it to cluster state
	if !initialized {
		logger.Info("node is not initialized yet, skipping", "node", instance.Name)
		return ctrl.Result{}, nil
	}

	// Fetch pods assigned to the node and update state
	var podList v1.PodList
	if err = c.Client.List(ctx, &podList, client.MatchingFields{constant.PodNodeNameKey: instance.Name}); err != nil {
		logger.Error(err, "unable to list pods assigned to node")
		return ctrl.Result{}, err
	}
	logger.V(2).Info("updating node", "node", instance.Name, "nPods", len(podList.Items))
	c.clusterState.UpdateNode(instance, podList.Items)

	return ctrl.Result{}, nil
}

func (c *NodeController) SetupWithManager(mgr ctrl.Manager, name string) error {
	// Reconcile only nodes with GPU partitioning enabled
	selectorPredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      v1alpha1.LabelGpuPartitioning,
			Operator: metav1.LabelSelectorOpExists,
		}},
	})
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1.Node{}, builder.WithPredicates(selectorPredicate)).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(c)
}
