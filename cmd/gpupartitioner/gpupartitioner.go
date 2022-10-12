package main

import (
	"flag"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/state"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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
	// Setup options
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Setup controller manager
	options := ctrl.Options{
		Scheme: scheme,
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clusterState := state.NewClusterState()

	// Setup state controllers
	nodeController := state.NewNodeController(
		mgr.GetClient(),
		mgr.GetScheme(),
		&clusterState,
	)
	if err := nodeController.SetupWithManager(mgr, constant.ClusterStateNodeControllerName); err != nil {
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
	if err := podController.SetupWithManager(mgr, constant.ClusterStatePodControllerName); err != nil {
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
	migPlanner, err := mig.NewPlanner(k8sClient)
	if err != nil {
		setupLog.Error(err, "unable to create MIG planner")
		os.Exit(1)
	}
	migActuator := mig.NewActuator()
	migController := core.NewController(
		mgr.GetClient(),
		mgr.GetScheme(),
		&clusterState,
		migPlanner,
		migActuator,
	)
	if err := migController.SetupWithManager(mgr, constant.MigPartitionerControllerName); err != nil {
		setupLog.Error(
			err,
			"unable to create controller",
			"controller",
			constant.MigPartitionerControllerName,
		)
		os.Exit(1)
	}

	// Setup health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start controller manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
