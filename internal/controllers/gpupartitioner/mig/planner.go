package mig

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/gpupartitioner/state"
	"github.com/nebuly-ai/nebulnetes/pkg/gpu/mig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	scheduler_config "k8s.io/kubernetes/pkg/scheduler/apis/config/latest"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	scheduler_plugins "k8s.io/kubernetes/pkg/scheduler/framework/plugins"
	"k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Planner struct {
	schedulerFramework framework.Framework
	logger             logr.Logger
}

func NewPlanner(kubeClient kubernetes.Interface, logger logr.Logger) (*Planner, error) {
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

	return &Planner{schedulerFramework: f, logger: logger}, nil
}

func (p Planner) newLogger(ctx context.Context) klog.Logger {
	return log.FromContext(ctx).WithName("MigPlanner")
}

func (p Planner) Plan(ctx context.Context, snapshot state.ClusterSnapshot, candidates []v1.Pod) (state.DesiredPartitioning, error) {
	logger := p.newLogger(ctx)
	res := snapshot.GetCurrentPartitioning()
	for _, pod := range candidates {
		lackingMig, isLacking := p.getLackingMigProfile(snapshot, pod)
		if !isLacking {
			continue
		}
		candidateNodes := p.getCandidateNodes(snapshot)
		logger.V(1).Info(
			fmt.Sprintf("found %d candidate nodes for pod", len(candidateNodes)),
			"namespace",
			pod.GetNamespace(),
			"pod",
			pod.GetName(),
			"lackingResource",
			lackingMig,
		)
		for _, n := range candidateNodes {
			// Check if node can potentially host the Pod by updating its MIG geometry
			if err := n.UpdateGeometryFor(lackingMig, 1); err != nil {
				continue
			}

			// Fork the state and update the nodes' allocatable scalar resources by taking into
			// account the new MIG geometry
			snapshot.Fork()
			nodeInfo, _ := snapshot.GetNode(n.Name)
			scalarResources := getUpdatedScalarResources(nodeInfo, n)
			nodeInfo.Allocatable.ScalarResources = scalarResources
			snapshot.SetNode(nodeInfo)

			// Run a scheduler cycle to check whether the Pod can be scheduled on the Node
			podFits, err := p.podFitsNode(ctx, nodeInfo, pod)
			if err != nil {
				return res, err
			}

			// The Pod cannot be scheduled, revert the changes on the snapthot
			if !podFits {
				snapshot.Revert()
				continue
			}

			// The Pod can be scheduled, commit changes
			if err = snapshot.AddPod(n.Name, pod); err != nil {
				return res, err
			}
			snapshot.Commit()

			// Update desired partitioning
			nodePartitioning := state.NodePartitioning{
				GPUs: make([]state.GPUPartitioning, 0),
			}
			for _, g := range n.GPUs {
				gpuPartitioning := state.GPUPartitioning{
					GPUIndex:  g.GetIndex(),
					Resources: g.GetGeometry().AsResourceList(),
				}
				nodePartitioning.GPUs = append(nodePartitioning.GPUs, gpuPartitioning)
			}
			res[n.Name] = nodePartitioning
		}
	}
	return res, nil
}

// getUpdatedScalarResources returns the scalar resources of the nodeInfo provided as argument updated for
// matching the MIG resources defied by the specified mig.Node
func getUpdatedScalarResources(nodeInfo framework.NodeInfo, node mig.Node) map[v1.ResourceName]int64 {
	res := make(map[v1.ResourceName]int64)

	// Set all non-MIG scalar resources
	for r, v := range nodeInfo.Allocatable.ScalarResources {
		if !mig.IsNvidiaMigDevice(r) {
			res[r] = v
		}
	}
	// Set MIG scalar resources
	for r, v := range node.GetGeometry().AsResourceList() {
		nodeInfo.Allocatable.ScalarResources[r] = v.Value()
	}

	return res
}

// getLackingMigProfile returns (if any) the MIG profile requested by the Pod but currently not
// available in the ClusterSnapshot.
//
// As described in "Supporting MIG GPUs in Kubernetes" document, it is assumed that
// Pods request only one MIG device per time and with quantity 1, according to the
// idea that users should ask for a larger, single instance as opposed to multiple
// smaller instances.
func (p Planner) getLackingMigProfile(snapshot state.ClusterSnapshot, pod v1.Pod) (mig.ProfileName, bool) {
	for r := range snapshot.GetLackingResources(pod).ScalarResources {
		if mig.IsNvidiaMigDevice(r) {
			profileName, _ := mig.ExtractMigProfile(r)
			return profileName, true
		}
	}
	return "", false
}

// getCandidateNodes returns the Nodes of the ClusterSnapshot with free (e.g. not allocated) MIG resources
// candidate for a MIG geometry updated aimed to schedule a pending pod
func (p Planner) getCandidateNodes(snapshot state.ClusterSnapshot) []mig.Node {
	result := make([]mig.Node, 0)

	var migNode mig.Node
	var err error

	for _, n := range snapshot.GetNodes() {
		if migNode, err = mig.NewNode(*n.Node()); err != nil {
			p.logger.Error(
				err,
				"unable to create MIG node",
				"node",
				n.Node().Name,
			)
			continue
		}
		if migNode.HasFreeMigResources() {
			result = append(result, migNode)
		}
	}

	return result
}

func (p Planner) podFitsNode(ctx context.Context, node framework.NodeInfo, pod v1.Pod) (bool, error) {
	cycleState := framework.NewCycleState()
	_, preFilterStatus := p.schedulerFramework.RunPreFilterPlugins(ctx, cycleState, &pod)
	if !preFilterStatus.IsSuccess() {
		return false, fmt.Errorf("error running pre filter plugins for pod %s; %s", pod.Name, preFilterStatus.Message())
	}
	filterStatus := p.schedulerFramework.RunFilterPlugins(ctx, cycleState, &pod, &node).Merge()
	return filterStatus.IsSuccess(), nil
}
