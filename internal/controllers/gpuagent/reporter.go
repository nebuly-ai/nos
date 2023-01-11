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

package gpuagent

import (
	"context"
	"github.com/nebuly-ai/nos/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/gpu"
	"github.com/nebuly-ai/nos/pkg/gpu/slicing"
	"github.com/nebuly-ai/nos/pkg/util/predicate"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

type Reporter struct {
	client.Client
	gpuClient       gpu.Client
	refreshInterval time.Duration
}

func NewReporter(k8sClient client.Client, gpuClient gpu.Client, refreshInterval time.Duration) Reporter {
	return Reporter{
		Client:          k8sClient,
		gpuClient:       gpuClient,
		refreshInterval: refreshInterval,
	}
}

//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch

func (r *Reporter) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := klog.FromContext(ctx)

	// Fetch node and get last status
	var instance v1.Node
	if err := r.Client.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &instance); err != nil {
		return ctrl.Result{}, err
	}
	lastStatusAnnotations, _ := gpu.ParseNodeAnnotations(instance)

	// Fetch GPUs
	devices, err := r.gpuClient.GetDevices(ctx)
	if err != nil {
		logger.Error(err, "unable to fetch GPUs")
		return ctrl.Result{}, err
	}

	// Check if status changed
	currentStatusAnnotations := devices.AsStatusAnnotation(slicing.ExtractProfileNameStr)
	logger.Info("computed annotations", "current", currentStatusAnnotations, "last", lastStatusAnnotations, "devices", devices)
	if currentStatusAnnotations.Equal(lastStatusAnnotations) {
		logger.Info("current status is equal to last reported status, nothing to do")
		return ctrl.Result{RequeueAfter: r.refreshInterval}, nil
	}

	// Update node
	logger.Info("status changed - reporting it by updating node annotations")
	updated := instance.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = make(map[string]string)
	}
	for k := range updated.Annotations {
		if strings.HasPrefix(k, v1alpha1.AnnotationGpuStatusPrefix) {
			delete(updated.Annotations, k)
		}
	}
	for _, a := range currentStatusAnnotations {
		updated.Annotations[a.String()] = a.GetValue()
	}
	if err := r.Client.Patch(ctx, updated, client.MergeFrom(&instance)); err != nil {
		logger.Error(err, "unable to update node status annotations", "annotations", updated.Annotations)
		return ctrl.Result{}, err
	}
	logger.Info("updated reported status - node annotations updated successfully")

	return ctrl.Result{RequeueAfter: r.refreshInterval}, nil
}

func (r *Reporter) SetupWithManager(mgr ctrl.Manager, controllerName string, nodeName string) error {
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
