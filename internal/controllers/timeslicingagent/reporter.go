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
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	"github.com/nebuly-ai/nebulnetes/pkg/util/predicate"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Reporter struct {
	client.Client
	gpuClient       gpu.Client
	refreshInterval time.Duration
}

func NewReporter(client client.Client, tsClient gpu.Client, refreshInterval time.Duration) Reporter {
	return Reporter{
		Client:          client,
		gpuClient:       tsClient,
		refreshInterval: refreshInterval,
	}
}

//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;patch

func (r *Reporter) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := klog.FromContext(ctx)

	// Fetch GPUs
	devices, err := r.gpuClient.GetDevices(ctx)
	if err != nil {
		logger.Error(err, "unable to fetch GPUs")
		return ctrl.Result{}, err
	}
	logger.Info("", devices)

	// Group by GPU index

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
