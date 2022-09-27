package mig

import (
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/state"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func NewController(client client.Client, scheme *runtime.Scheme, clusterState *state.ClusterState) core.Controller {
	restConfig := config.GetConfigOrDie()
	k8sClient := kubernetes.NewForConfigOrDie(restConfig)
	migPartitioner := NewPartitioner(clusterState, client, k8sClient)
	return core.NewController(client, scheme, clusterState, migPartitioner)
}
