package mig

import (
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/core"
	"github.com/nebuly-ai/nebulnetes/pkg/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/util/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	scheduler_config "k8s.io/kubernetes/pkg/scheduler/apis/config/latest"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	scheduler_plugins "k8s.io/kubernetes/pkg/scheduler/framework/plugins"
	"k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

type Planner struct {
	snapshot           state.ClusterSnapshot
	schedulerFramework framework.Framework
}

func NewPlanner(kubeClient kubernetes.Interface) (*Planner, error) {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
	config, err := scheduler_config.Default()
	if err != nil {
		return nil, fmt.Errorf("couldn't create scheduler config: %v", err)
	}
	if len(config.Profiles) != 1 || config.Profiles[0].SchedulerName != v1.DefaultSchedulerName {
		return nil, fmt.Errorf("unexpected scheduler config: expected default scheduler profile only (found %d profiles)", len(config.Profiles))
	}

	f, err := runtime.NewFramework(
		scheduler_plugins.NewInTreeRegistry(),
		&config.Profiles[0],
		runtime.WithInformerFactory(informerFactory),
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't create scheduler framework; %v", err)
	}

	return &Planner{schedulerFramework: f}, nil
}

func (p Planner) GetNodesPartitioningPlan(pendingCandidates []v1.Pod) (map[string]core.PartitioningPlan, error) {
	plan := make(map[string]core.PartitioningPlan)
	for _, pod := range pendingCandidates {
		lackingMIGs := p.getLackingMIGResources(pod)
		if len(lackingMIGs) == 0 {
			continue
		}
		// TODO
	}
	return plan, nil
}

func (p Planner) getLackingMIGResources(pod v1.Pod) v1.ResourceList {
	result := make(v1.ResourceList)
	for r, q := range p.snapshot.GetLackingResources(pod) {
		if resource.IsNvidiaMigDevice(r) {
			result[r] = q
		}
	}
	return result
}
