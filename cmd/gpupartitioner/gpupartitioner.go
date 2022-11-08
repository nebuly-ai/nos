package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/mig"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	testutil "github.com/nebuly-ai/nebulnetes/pkg/test/util"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	scheduler_config "k8s.io/kubernetes/pkg/scheduler/apis/config/latest"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	scheduler_plugins "k8s.io/kubernetes/pkg/scheduler/framework/plugins"
	scheduler_runtime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
}

func main() {
	// Setup CLI args
	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var err error

	// Setup controller manager
	options := ctrl.Options{
		Scheme: scheme,
	}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	ctx := ctrl.SetupSignalHandler()

	clusterState := state.NewClusterState()

	// Setup indexer
	if err = setupIndexer(ctx, mgr); err != nil {
		setupLog.Error(err, "error configuring controller manager indexer")
		os.Exit(1)
	}

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

	// Setup MIG partitioner controller
	k8sClient := kubernetes.NewForConfigOrDie(config.GetConfigOrDie())
	schedulerFramework, err := newSchedulerFramework(k8sClient)
	if err != nil {
		setupLog.Error(err, "unable to init k8s scheduler framework")
		os.Exit(1)
	}
	migPlanner := mig.NewPlanner(schedulerFramework)
	if err != nil {
		setupLog.Error(err, "unable to create MIG planner")
		os.Exit(1)
	}
	podBatcher := util.NewBatcher[v1.Pod](1*time.Minute, 5*time.Second) // TODO move to config
	go func() {
		if err = podBatcher.Start(ctx); err != nil {
			setupLog.Error(err, "unable to start pod batcher")
			os.Exit(1)
		}
	}()
	migActuator := mig.NewActuator(mgr.GetClient())
	migController := core.NewController(
		mgr.GetScheme(),
		mgr.GetClient(),
		podBatcher,
		&clusterState,
		migPlanner,
		migActuator,
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

func newSchedulerFramework(kubeClient kubernetes.Interface) (framework.Framework, error) {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
	schedulerConfig, err := scheduler_config.Default()
	if err != nil {
		return nil, fmt.Errorf("couldn't create scheduler config: %v", err)
	}
	if len(schedulerConfig.Profiles) != 1 || schedulerConfig.Profiles[0].SchedulerName != v1.DefaultSchedulerName {
		return nil, fmt.Errorf(
			"unexpected scheduler config: expected default scheduler profile only (found %d profiles)",
			len(schedulerConfig.Profiles),
		)
	}
	return scheduler_runtime.NewFramework(
		scheduler_plugins.NewInTreeRegistry(),
		&schedulerConfig.Profiles[0],
		scheduler_runtime.WithInformerFactory(informerFactory),
		scheduler_runtime.WithSnapshotSharedLister(testutil.NewFakeSharedLister(make([]*v1.Pod, 0), make([]*v1.Node, 0))),
	)
}
