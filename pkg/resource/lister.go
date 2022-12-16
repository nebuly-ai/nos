package resource

import (
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
	"time"
)

const (
	// PodResourcesPath is the path to the local endpoint serving the PodResources GRPC service.
	PodResourcesPath = "/var/lib/kubelet/pod-resources"
)

func NewPodResourcesListerClient(timeout time.Duration, maxMsgSize int) (pdrv1.PodResourcesListerClient, error) {
	endpoint, err := util.LocalEndpoint(PodResourcesPath, podresources.Socket)
	if err != nil {
		return nil, err
	}
	listerClient, _, err := podresources.GetV1Client(endpoint, timeout, maxMsgSize)
	return listerClient, err
}
