//go:build nvml

package main

import (
	"flag"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/migagent"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
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

const (
	// defaultPodResourcesPath is the path to the local endpoint serving the PodResources GRPC service.
	defaultPodResourcesPath    = "/var/lib/kubelet/pod-resources"
	defaultPodResourcesTimeout = 10 * time.Second
	defaultPodResourcesMaxSize = 1024 * 1024 * 16 // 16 Mb
)

func main() {
	// Setup options
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
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Init MIG client
	podResourcesClient, err := newPodResourcesListerClient()
	setupLog.Info("Initializing NVML client")
	nvmlClient, err := nvml.NewClient()
	if err != nil {
		setupLog.Error(err, "unable to init nvml client")
		os.Exit(1)
	}
	migClient := mig.NewClient(podResourcesClient, nvmlClient)

	// Setup MIG Reporter
	migReporter := migagent.NewReporter(
		mgr.GetClient(),
		&migClient,
		10*time.Second,
	)
	if err := migReporter.SetupWithManager(mgr, "MIGReporter", nodeName); err != nil {
		setupLog.Error(err, "unable to create MIG Reporter")
		os.Exit(1)
	}

	// Add health check endpoints to manager
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start manager
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func newSharedFactoryForNode(k8sClient *kubernetes.Clientset, nodeName string) informers.SharedInformerFactory {
	return informers.NewSharedInformerFactoryWithOptions(
		k8sClient,
		0,
		informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fmt.Sprintf("metadata.name=%s", nodeName)
		}),
	)
}

func newPodResourcesListerClient() (pdrv1.PodResourcesListerClient, error) {
	endpoint, err := util.LocalEndpoint(defaultPodResourcesPath, podresources.Socket)
	if err != nil {
		return nil, err
	}
	client, _, err := podresources.GetV1Client(endpoint, defaultPodResourcesTimeout, defaultPodResourcesMaxSize)
	return client, err
}
