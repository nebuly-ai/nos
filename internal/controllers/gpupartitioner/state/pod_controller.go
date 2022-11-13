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

type PodController struct {
	client.Client
	Scheme       *runtime.Scheme
	clusterState *ClusterState
}

func NewPodController(client client.Client, scheme *runtime.Scheme, state *ClusterState) PodController {
	return PodController{
		Client:       client,
		Scheme:       scheme,
		clusterState: state,
	}
}

func (c *PodController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch instance
	var instance v1.Pod
	objKey := client.ObjectKey{Namespace: req.Namespace, Name: req.Name}
	err := c.Client.Get(ctx, objKey, &instance)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to fetch pod")
		return ctrl.Result{}, err
	}

	// If Pod does not exist then remove it from Cluster State
	if apierrors.IsNotFound(err) {
		logger.V(1).Info("deleting pod", "pod", req.Name, "namespace", req.Namespace)
		_ = c.clusterState.deletePod(req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// If Pod is not assigned to any node then skip state update
	nodeName := instance.Spec.NodeName
	if nodeName == "" {
		logger.V(1).Info(
			"pod is not assigned to any node, skipping cluster state update",
			"pod",
			req.Name,
			"namespace",
			req.Namespace,
		)
		return ctrl.Result{}, nil
	}

	// If node does not exist already exists in cluster state we need to add it
	if _, found := c.clusterState.GetNode(nodeName); !found {
		logger.V(1).Info("pod's node not found in cluster state", "node", nodeName)
		var podNode v1.Node
		nodeKey := client.ObjectKey{Namespace: "", Name: nodeName}
		if err = c.Client.Get(ctx, nodeKey, &podNode); err != nil {
			if apierrors.IsNotFound(err) {
				logger.V(1).Info("node does not exist anymore, removing it from state", "node", nodeName)
				c.clusterState.deleteNode(nodeName)
			}
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		var podList v1.PodList
		if err = c.Client.List(ctx, &podList, client.MatchingFields{constant.PodNodeNameKey: nodeName}); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		logger.V(1).Info("adding node", "node", nodeName)
		c.clusterState.updateNode(podNode, podList.Items)

		return ctrl.Result{}, nil
	}

	logger.V(1).Info("updating cluster state usage", "pod", req.Name, "namespace", req.Namespace)
	c.clusterState.updateUsage(instance)
	return ctrl.Result{}, nil
}

func (c *PodController) SetupWithManager(mgr ctrl.Manager, name string) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1.Pod{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(c)
}
