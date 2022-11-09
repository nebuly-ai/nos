//go:build nvml

package main

import (
	"context"
	"flag"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/migagent"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sync"
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

const (
	// defaultPodResourcesPath is the path to the local endpoint serving the PodResources GRPC service.
	defaultPodResourcesPath    = "/var/lib/kubelet/pod-resources"
	defaultPodResourcesTimeout = 10 * time.Second
	defaultPodResourcesMaxSize = 1024 * 1024 * 16 // 16 Mb
)

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

	// Get node name
	nodeName, err := util.GetEnvOrError("NODE_NAME")
	if err != nil {
		setupLog.Error(err, "missing required env variable")
		os.Exit(1)
	}

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

	// Setup indexer
	err = mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Pod{}, constant.PodNodeNameKey, func(rawObj client.Object) []string {
		p := rawObj.(*v1.Pod)
		return []string{p.Spec.NodeName}
	})
	if err != nil {
		setupLog.Error(err, "unable to configure indexer")
		os.Exit(1)
	}

	// mutex for reporter/actuator synchronization
	var mutex sync.Mutex

	// Init MIG client
	podResourcesClient, err := newPodResourcesListerClient()
	setupLog.Info("Initializing NVML client")
	nvmlClient := nvml.NewClient(ctrl.Log.WithName("NvmlClient"))
	migClient := mig.NewClient(podResourcesClient, nvmlClient)

	// Setup MIG Reporter
	migReporter := migagent.NewReporter(
		mgr.GetClient(),
		migClient,
		&mutex,
		10*time.Second,
	)
	if err = migReporter.SetupWithManager(mgr, "MIGReporter", nodeName); err != nil {
		setupLog.Error(err, "unable to create MIG Reporter")
		os.Exit(1)
	}

	// Setup MIG Actuator
	migActuator := migagent.NewActuator(
		mgr.GetClient(),
		migClient,
		&mutex,
		nodeName,
	)
	if err = migActuator.SetupWithManager(mgr, "MIGActuator"); err != nil {
		setupLog.Error(err, "unable to create MIG Actuator")
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
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func newPodResourcesListerClient() (pdrv1.PodResourcesListerClient, error) {
	endpoint, err := util.LocalEndpoint(defaultPodResourcesPath, podresources.Socket)
	if err != nil {
		return nil, err
	}
	client, _, err := podresources.GetV1Client(endpoint, defaultPodResourcesTimeout, defaultPodResourcesMaxSize)
	return client, err
}
