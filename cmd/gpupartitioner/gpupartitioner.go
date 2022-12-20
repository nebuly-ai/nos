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
	"context"
	"flag"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/mig"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	configv1alpha1 "github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/config/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/api/scheduler"
	schedulerv1beta3 "github.com/nebuly-ai/nebulnetes/pkg/api/scheduler/v1beta3"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	gpumig "github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/scheduler/plugins/capacityscheduling"
	testutil "github.com/nebuly-ai/nebulnetes/pkg/test/util"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	schedulerconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"
	latestschedulerconfig "k8s.io/kubernetes/pkg/scheduler/apis/config/latest"
	schedulerscheme "k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	schedulerplugins "k8s.io/kubernetes/pkg/scheduler/framework/plugins"
	schedulerruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
	// Ensure scheduler package is initialized.
	_ "github.com/nebuly-ai/nebulnetes/pkg/api/scheduler"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(scheduler.AddToScheme(scheme))
	utilruntime.Must(schedulerv1beta3.AddToScheme(scheme))
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

	var err error

	// Load config
	options := ctrl.Options{
		Scheme: scheme,
	}
	config := configv1alpha1.GpuPartitionerConfig{}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&config))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}
	if err = config.Validate(); err != nil {
		setupLog.Error(err, "config is invalid")
		os.Exit(1)
	}

	// Setup known MIG geometries
	if config.KnownMigGeometriesFile != "" {
		knownGeometries, err := loadKnownGeometriesFromFile(config.KnownMigGeometriesFile)
		if err != nil {
			setupLog.Error(err, "unable to load known MIG geometries")
			os.Exit(1)
		}
		if err = gpumig.SetKnownGeometries(knownGeometries); err != nil {
			setupLog.Error(err, "unable to set known MIG geometries")
			os.Exit(1)
		}
		setupLog.Info("using known MIG geometries loaded from file", "geometries", knownGeometries)
	}

	// Setup controller manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	ctx := ctrl.SetupSignalHandler()

	// Setup indexer
	if err = setupIndexer(ctx, mgr); err != nil {
		setupLog.Error(err, "error configuring controller manager indexer")
		os.Exit(1)
	}

	// Init state
	clusterState := state.NewEmptyClusterState()

	// Setup state controllers
	nodeController := state.NewNodeController(
		mgr.GetClient(),
		mgr.GetScheme(),
		&clusterState,
	)
	if err = nodeController.SetupWithManager(mgr, constant.ClusterStateNodeControllerName); err != nil {
		setupLog.Error(
			err,
			"unable to create controller",
			"controller",
			constant.ClusterStateNodeControllerName,
		)
		os.Exit(1)
	}
	podController := state.NewPodController(
		mgr.GetClient(),
		mgr.GetScheme(),
		&clusterState,
	)
	if err = podController.SetupWithManager(mgr, constant.ClusterStatePodControllerName); err != nil {
		setupLog.Error(
			err,
			"unable to create controller",
			"controller",
			constant.ClusterStatePodControllerName,
		)
		os.Exit(1)
	}

	// Init scheduler and planner
	k8sClient := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
	schedulerFramework, err := newSchedulerFramework(ctx, config, k8sClient)
	if err != nil {
		setupLog.Error(err, "unable to init k8s scheduler framework")
		os.Exit(1)
	}
	migPlanner := mig.NewPlanner(schedulerFramework)
	if err != nil {
		setupLog.Error(err, "unable to create MIG planner")
		os.Exit(1)
	}

	// Init and start Pods batcher
	windowTimeoutDuration := config.BatchWindowTimeoutSeconds * time.Second
	windowIdleDuration := config.BatchWindowIdleSeconds * time.Second
	podBatcher := util.NewBatcher[v1.Pod](
		windowTimeoutDuration,
		windowIdleDuration,
	)
	setupLog.Info(
		"pods batch window",
		"timeout",
		windowTimeoutDuration.String(),
		"idle",
		windowIdleDuration.String(),
	)
	go func() {
		if err = podBatcher.Start(ctx); err != nil {
			setupLog.Error(err, "unable to start pod batcher")
			os.Exit(1)
		}
	}()

	// Init actuator
	migActuator := mig.NewActuator(mgr.GetClient())

	// Setup MIG controller
	migController := core.NewController(
		mgr.GetScheme(),
		mgr.GetClient(),
		podBatcher,
		&clusterState,
		migPlanner,
		migActuator,
		mig.NewSnapshotTaker(),
	)
	if err = migController.SetupWithManager(mgr, constant.MigPartitionerControllerName); err != nil {
		setupLog.Error(
			err,
			"unable to create controller",
			"controller",
			constant.MigPartitionerControllerName,
		)
		os.Exit(1)
	}

	// Setup health checks
	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start controller manager
	setupLog.Info("starting manager")
	if err = mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupIndexer(ctx context.Context, mgr ctrl.Manager) error {
	var err error

	// Index Pods' phase
	err = mgr.GetFieldIndexer().IndexField(ctx, &v1.Pod{}, constant.PodPhaseKey, func(rawObj client.Object) []string {
		p := rawObj.(*v1.Pod)
		return []string{string(p.Status.Phase)}
	})
	if err != nil {
		return err
	}

	// Index Pods' node name
	err = mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Pod{}, constant.PodNodeNameKey, func(rawObj client.Object) []string {
		p := rawObj.(*v1.Pod)
		return []string{p.Spec.NodeName}
	})
	if err != nil {
		return err
	}

	return nil
}

func newSchedulerFramework(ctx context.Context, config configv1alpha1.GpuPartitionerConfig, kubeClient kubernetes.Interface) (framework.Framework, error) {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)

	// Configure scheduler profile
	profile, err := getSchedulerProfile(config)
	if err != nil {
		return nil, err
	}
	setupLog.V(1).Info("scheduler profile", "profile", profile)

	// Register capacity scheduling plugin
	var registry = schedulerplugins.NewInTreeRegistry()
	if err = registry.Register(capacityscheduling.Name, capacityscheduling.New); err != nil {
		return nil, fmt.Errorf("couldn't register Capacity Scheduling plugin: %v", err)
	}

	return schedulerruntime.NewFramework(
		registry,
		&profile,
		ctx.Done(),
		schedulerruntime.WithInformerFactory(informerFactory),
		schedulerruntime.WithKubeConfig(ctrl.GetConfigOrDie()),
		schedulerruntime.WithSnapshotSharedLister(testutil.NewFakeSharedLister(make([]*v1.Pod, 0), make([]*v1.Node, 0))),
	)
}

func getSchedulerProfile(config configv1alpha1.GpuPartitionerConfig) (schedulerconfig.KubeSchedulerProfile, error) {
	// If scheduler config is not provided, use default scheduler config
	if config.SchedulerConfigFile == "" {
		defaultSchedulerConfig, err := latestschedulerconfig.Default()
		if err != nil {
			return schedulerconfig.KubeSchedulerProfile{}, fmt.Errorf("couldn't create scheduler config: %v", err)
		}
		if len(defaultSchedulerConfig.Profiles) != 1 || defaultSchedulerConfig.Profiles[0].SchedulerName != v1.DefaultSchedulerName {
			return schedulerconfig.KubeSchedulerProfile{}, fmt.Errorf(
				"unexpected scheduler config: expected default scheduler profile only (found %d profiles)",
				len(defaultSchedulerConfig.Profiles),
			)
		}
		setupLog.Info("scheduler configured with default profile")
		return defaultSchedulerConfig.Profiles[0], nil
	}

	// Otherwise, use the scheduler config provided in the GpuPartitionerConfig
	schedulerConfig, err := loadSchedulerConfigFromFile(config.SchedulerConfigFile)
	if err != nil {
		return schedulerconfig.KubeSchedulerProfile{}, fmt.Errorf(
			"couldn't load scheduler config: %v",
			err,
		)
	}
	profile := schedulerConfig.Profiles[0]
	setupLog.Info("scheduler configured with custom profile")
	return profile, nil
}

func loadSchedulerConfigFromFile(file string) (*schedulerconfig.KubeSchedulerConfiguration, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return decodeSchedulerConfig(data)
}

func decodeSchedulerConfig(data []byte) (*schedulerconfig.KubeSchedulerConfiguration, error) {
	// The UniversalDecoder runs defaulting and returns the internal type by default.
	obj, gvk, err := schedulerscheme.Codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}
	if cfgObj, ok := obj.(*schedulerconfig.KubeSchedulerConfiguration); ok {
		return cfgObj, nil
	}
	return nil, fmt.Errorf("couldn't decode as KubeSchedulerConfiguration, got %s: ", gvk)
}

func loadKnownGeometriesFromFile(file string) (map[gpu.Model][]gpumig.Geometry, error) {
	var knownGeometries = make(map[gpu.Model][]gpumig.Geometry)
	data, err := os.ReadFile(file)
	if err != nil {
		return knownGeometries, err
	}
	err = yaml.Unmarshal(data, &knownGeometries)
	return knownGeometries, err
}
