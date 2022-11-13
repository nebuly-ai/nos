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

package state

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NodeController struct {
	client.Client
	Scheme       *runtime.Scheme
	clusterState *ClusterState
}

func NewNodeController(client client.Client, scheme *runtime.Scheme, state *ClusterState) NodeController {
	return NodeController{
		Client:       client,
		Scheme:       scheme,
		clusterState: state,
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
		logger.V(1).Info("deleting node", "node", instance.Name)
		c.clusterState.deleteNode(instance.Name)
		return ctrl.Result{}, nil
	}

	// Fetch pods assigned to the node and update state
	var podList v1.PodList
	if err = c.Client.List(ctx, &podList, client.MatchingFields{constant.PodNodeNameKey: instance.Name}); err != nil {
		logger.Error(err, "unable to list pods assigned to node")
		return ctrl.Result{}, err
	}
	logger.V(1).Info("updating node", "node", instance.Name, "nPods", len(podList.Items))
	c.clusterState.updateNode(instance, podList.Items)

	return ctrl.Result{}, nil
}

func (c *NodeController) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1.Node{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(c)
}
