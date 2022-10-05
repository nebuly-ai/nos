//go:build nvml

package main

import (
	"context"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/components/mighandler"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/nvml"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"time"
)

const (
	// defaultPodResourcesPath is the path to the local endpoint serving the PodResources GRPC service.
	defaultPodResourcesPath    = "/var/lib/kubelet/pod-resources"
	defaultPodResourcesTimeout = 10 * time.Second
	defaultPodResourcesMaxSize = 1024 * 1024 * 16 // 16 Mb
)

func main() {
	ctx := context.Background()
	logger := klog.FromContext(ctx).WithName("setup")
	k8sClient := kubernetes.NewForConfigOrDie(config.GetConfigOrDie())

	// Init shared factory
	nodeName, err := util.GetEnvOrError("NODE_NAME")
	if err != nil {
		logger.Error(err, "missing required env variable")
		os.Exit(1)
	}
	sharedFactory := newSharedFactoryForNode(k8sClient, nodeName)

	// Init MIG Reporter
	podResourcesClient, err := newPodResourcesListerClient()
	migClient := mig.NewClient(podResourcesClient, nvml.NewClient())
	migReporter := mighandler.NewMIGReporter(
		nodeName,
		k8sClient,
		migClient,
		sharedFactory,
		10*time.Second,
	)

	// Start MIG Reporter
	if err := migReporter.Start(ctx); err != nil {
		logger.Error(err, "unable to start MIG Reporter", "node", nodeName)
		os.Exit(1)
	}

	<-ctx.Done()
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
