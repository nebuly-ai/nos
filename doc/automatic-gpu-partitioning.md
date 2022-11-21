# Automatic GPU partitioning

> ⚠️ At the moment Nebulnetes only
> supports [Multi-instance GPU (MIG) partitioning](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/index.html),
> which is available only for NVIDIA GPUs based on Ampere and Hopper architectures.

## Overview

The automatic GPU partitioning is performed by the [GPU Partitioner](../config/gpupartitioner) component, which
constantly watches the GPU resources of the cluster and finds the best possible partitioning of the available GPUs
in order to schedule the highest number of pods requesting fractions of GPUs, which otherwise could not be scheduled
due to the lack of available resources.

You can see the GPU Partitioner as a sort of [Cluster Autoscaler](https://github.com/kubernetes/autoscaler) for GPUs:
instead of scaling up the number of nodes and GPUs, it dynamically partitions the available GPUs to maximize
their utilization.

The actual partitioning of the GPUs is not performed directly by the GPU Partitioner, but instead it is carried out
by an agent deployed on every node of the cluster eligible for automatic partitioning. This agent exposes
to the GPU Partitioner the partitioning state of the GPUs of the node on which it is running, and applies the desired
partitioning state decided by the GPU Partitioner.

### MIG partitioning

In the case of MIG partitioning, the agent that creates/deletes the MIG resources is the [MIG Agent](../config/migagent)
,
which is a daemonset running on every node labeled with `n8s.nebuly.ai/auto-mig-enabled: "true"`.

The MIG Agent exposes to the GPU Partitioner the used/free MIG resources of all the GPUs of the node
on which it is running through the following node annotations:

* `n8s.nebuly.ai/status-gpu-<index>-<mig-profile>-free: <quantity>`
* `n8s.nebuly.ai/status-gpu-<index>-<mig-profile>-used: <quantity>`

The MIG Agent also watches the node's annotations and, every time there desired MIG partitioning specified by the
GPU Partitioner does not match the current state, it tries to apply it by creating and deleting the MIG profiles
on the target GPUs. The GPU Partitioner specifies the desired MIG geometry of the GPUs of a node through annotations in
the following format:

`n8s.nebuly.ai/spec-gpu-<index>-<mig-profile>: <quantity>`

Note that, before applying the desired status, the MIG Agent always checks whether the MIG resources that need to
be deleted are not currently used by any pod. If this is the case, the MIG Agent will not delete the MIG resources.

## Configuration

You can customize the GPU Partitioning settings by editing the
[GPU Partitioner configuration file](../config/gpupartitioner/manager/gpu_partitioner_config.yaml).

### Pods batch size

The GPU partitioner processes pending pods in batches of configurable size. You can set the batch size by editing the
following two parameters of the configuration:

* `batchWindowTimeoutSeconds`: timeout of the time window used for batching pending Pods. The time window starts
  when the GPU Partitioner starts processing a batch of pending Pods, and ends when the timeout expires or the
  batch is completed.
* `batchWindowIdleSeconds`: idle time before a batch of pods is considered completed. Once the time window of a batch
  starts, if idle time elapses and no new pending pods are detected during this time, the batch is considered completed.

Increase the value of these two parameters if you want the GPU partitioner to take into account more pending Pods
when deciding the GPU partitioning plan, thus making potentially it more effective.

Set lower values if you want the partitioning to be performed more frequently
(e.g. if you want to react faster to changes in the cluster), and you don't mind if the partitioning is less effective
(e.g. the resources requested by some pending pods might not be created).

### Scheduler configuration

The GPU Partitioner uses an internal scheduler to simulate the scheduling of the pending pods to determine whether
a candidate GPU partitioning plan would make the pending pods schedulable.

You can provide a custom scheduler configuration by editing the `schedulerConfigFile` parameter, which is the path
to a YAML file containing the scheduler configuration.

#### GPU Partitioning with Nebulnetes scheduler

If you want to use Automatic GPU partitioning together with the Nebulnetes scheduler so that Elastic Resource quotas
are taken into account when performing the GPUs partitioning, you can follow these steps:

* deploy the Nebulnetes scheduler
* uncomment the last patch of this [kustomization file](../config/gpupartitioner/default/kustomization.yaml), which
  mounts the n8s scheduler config file to the GPU partitioner pod filesystem
* set the `schedulerConfigFile` value to `scheduler_config.yam`