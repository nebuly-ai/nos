/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
)

// StaticTrainingJobReconciler reconciles a StaticTrainingJob object
type StaticTrainingJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=statictrainingjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=statictrainingjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=n8s.nebuly.ai,resources=statictrainingjobs/finalizers,verbs=update

func (r *StaticTrainingJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var staticTrainingJobs n8sv1alpha1.StaticTrainingJobList
	if err := r.List(ctx, &staticTrainingJobs); err != nil {
		logger.Error(err, "unable to list StaticTrainingJobs")
		return ctrl.Result{}, err
	}

	logger.Info(fmt.Sprintf("Fetched StaticTrainingJobs: %v", staticTrainingJobs))

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StaticTrainingJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&n8sv1alpha1.StaticTrainingJob{}).
		Complete(r)
}
