//go:build nvml

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
	"errors"
	"flag"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpuagent"
	configv1alpha1 "github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/config/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/slicing"
	"github.com/nebuly-ai/nebulnetes/pkg/resource"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(configv1alpha1.AddToScheme(scheme))
}

func main() {
	// Setup CLI args
	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ctx := ctrl.SetupSignalHandler()

	// Get node name
	nodeName, err := util.GetEnvOrError(constant.EnvVarNodeName)
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("missing required env variable %s", constant.EnvVarNodeName))
		os.Exit(1)
	}

	// Load config and setup controller manager
	options := ctrl.Options{
		Scheme: scheme,
	}
	agentConfig := configv1alpha1.GpuAgentConfig{}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&agentConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}
	reportingSeconds := agentConfig.ReportConfigIntervalSeconds * time.Second
	setupLog.Info("loaded config", "reportingInterval", reportingSeconds)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Init MIG client
	lister, err := resource.NewPodResourcesListerClient(
		constant.DefaultPodResourcesTimeout,
		constant.DefaultPodResourcesMaxMsgSize,
	)
	resourceClient := resource.NewClient(lister)
	setupLog.Info("Initializing NVML client")
	nvmlClient := nvml.NewClient(ctrl.Log.WithName("NvmlClient"))
	gpuClient := slicing.NewClient(resourceClient, nvmlClient)

	// Check if any of the GPUs of the node has MIG mode enabled
	anyMigEnabledGpu, err := AnyMigEnabledGpu(nvmlClient)
	if err != nil {
		setupLog.Error(err, "unable to fetch GPUs")
		os.Exit(1)
	}
	if anyMigEnabledGpu {
		setupLog.Error(errors.New("cannot run on a node with MIG enabled GPUs"), "exiting")
		os.Exit(1)
	}

	// Setup Reporter
	migReporter := gpuagent.NewReporter(
		mgr.GetClient(),
		gpuClient,
		reportingSeconds,
	)
	if err = migReporter.SetupWithManager(mgr, "reporter", nodeName); err != nil {
		setupLog.Error(err, "unable to create time-slicing Reporter")
		os.Exit(1)
	}

	// Add health check endpoints to manager
	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start manager
	setupLog.Info("starting manager")
	if err = mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// AnyMigEnabledGpu returns true if any of the GPUs of the node has MIG mode enabled
func AnyMigEnabledGpu(client nvml.Client) (bool, error) {
	migGpus, err := client.GetMigEnabledGPUs()
	if err != nil {
		return false, err
	}
	return len(migGpus) > 0, nil
}
