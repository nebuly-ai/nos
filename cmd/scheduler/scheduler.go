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

package main

import (
	"github.com/nebuly-ai/nos/pkg/api/scheduler"
	"github.com/nebuly-ai/nos/pkg/api/scheduler/v1beta3"
	"github.com/nebuly-ai/nos/pkg/scheduler/plugins/capacityscheduling"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
	kubeschedulerscheme "k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"

	// Ensure n8s.nebuly.ai/v1alpha1 package is initialized
	_ "github.com/nebuly-ai/nos/pkg/api/n8s.nebuly.ai/v1alpha1"
	// Ensure scheduler package is initialized.
	_ "github.com/nebuly-ai/nos/pkg/api/scheduler"
)

var (
	// Re-use the in-tree Scheme.
	scheme = kubeschedulerscheme.Scheme
)

func main() {
	rand.Seed(time.Now().UnixNano())

	utilruntime.Must(scheduler.AddToScheme(scheme))
	utilruntime.Must(v1beta3.AddToScheme(scheme))

	command := app.NewSchedulerCommand(
		app.WithPlugin(capacityscheduling.Name, capacityscheduling.New),
	)

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
