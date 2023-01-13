# Automatic GPU partitioning

## Table of contents
- [Overview](#overview)
- [Partitioning modes comparison](#partitioning-modes-comparison)
- [MIG partitioning](#mig-partitioning)
- [MPS partitioning](#mps-partitioning)
- [Configuration](#configuration)
  - [Pods batch size](#pods-batch-size)
  - [Scheduler configuration](#scheduler-configuration)
  - [Available MIG geometries](#available-mig-geometries)
- [Troubleshooting](#troubleshooting)

## Overview
`nos` allows you to schedule Pods requesting fractions of GPUs without having to manually partition them:
the partitioning is performed dynamically based on the pending and running Pods in your cluster, so that the GPUs
are always fully utilized.

The GPU partitioning is performed by the [GPU Partitioner](../helm-charts/nos-gpu-partitioner) component, which
constantly watches the GPU resources of the cluster and finds the best possible partitioning of the available GPUs
in order to schedule the highest number of pods requesting fractions of GPUs, which otherwise could not be scheduled
due to the lack of available resources.

You can see the GPU Partitioner as a sort of [Cluster Autoscaler](https://github.com/kubernetes/autoscaler) for GPUs:
instead of scaling up the number of nodes and GPUs, it dynamically partitions the available GPUs to maximize
their utilization. 

The GPU partitioning is performed either using 
[Multi-instance GPU (MIG)](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/) or 
[Multi-Process Service (MPS)](https://docs.nvidia.com/deploy/mps/index.html), depending on the partitioning mode 
you choose for each node. You can find more info about these two in the section below.

## Partitioning modes comparison

| Partitioning mode          | Supported by `nos` | Workload isolation level | Pros                                                                                                                            | Cons                                                                                                                                                        |
|----------------------------|:-------------------|:-------------------------|---------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Multi-instance GPU (MIG)   | ✅                  | Best                     | <ul><li>Processes are executed concurrently</li><li>Full isolation (dedicated memory and compute resources)</li></ul>           | <ul><li>Supported by fewer GPU models (only Ampere or more recent architectures)</li><li>Coarse-grained control over memory and compute resources</li></ul> |
| Multi-process server (MPS) | ✅                  | Good                     | <ul><li>Processes are executed concurrently</li><li>Fine-grained control over memory and compute resources allocation</li></ul> | <ul><li>No error isolation and memory protection</li></ul>                                                                                                  |
| Time-slicing               | ❌                  | None                     | <ul><li>Supported by older GPU architectures (Pascal or newer)</li></ul>                                                        | <ul><li>No resource limits</li><li>No memory isolation</li><li>Lower performance due to context-switching overhead</li></ul>                                |


### Multi-instance GPU (MIG)
todo
### Multi-Process Service (MPS)
todo




## MIG partitioning
### Prerequisites
> ⚠️ [Multi-instance GPU (MIG)](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/index.html) mode
> is supported only by NVIDIA GPUs based on Ampere and Hopper architectures.


* you need the [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) deployed on your cluster
  * MIG strategy must be set to `mixed` (`--set mig.strategy=mixed`)
  * mig-manager must be disabled (`--set migManager.enabled=false`)
* if a node has multiple GPUs, all the GPUs must be of the same model
* all the GPUs of the nodes for which you want to enable MIG partitioning must have MIG mode enabled
  * you can enable MIG mode by running the following command for each GPU want to enable,
    where `<index>` correspond to the index of the GPU: `sudo nvidia-smi -i <index> -mig 1`
  * depending on the kind of machine you are using, it may be necessary to reboot the node after enabling MIG mode 
  for one of its GPUs.

### Enable automatic MIG partitioning on a node

You can enable automatic MIG partitioning on a node by adding to it the following label:

```shell
kubectl label nodes <node-name> "nos.nebuly.ai/gpu-partitioning=mig"
```
The label delegates to `nos` the management of the MIG resources of all the GPUs of that node, so you don't have
to manually configure the MIG geometry of the GPUs anymore: `nos` will dynamically create and delete the MIG profiles 
according to the resources requested by the pods submitted to the cluster, and according to the possible MIG geometries
supported by the GPUs in the cluster.

The available MIG geometries supported by each GPU model are defined in a ConfigMap, which by default is filled 
with the supported geometries of the most popular GPU models. You can override the values of this 
ConfigMap through the `nos-gpu-partitioner.knownMigGeometries` value of the 
[nos-gpu-partitioner Helm chart](../helm-charts/nos-gpu-partitioner). For more information you can refer to the 
[configuration section](#available-mig-geometries) of this document.


### How it works
The actual partitioning for MIG GPUs is performed by MIG Agent, which is a daemonset running on every node labeled 
with `nos.nebuly.ai/gpu-partitioning: mig` that creates/deletes MIG profiles as requested by the GPU Partitioner.

The MIG Agent exposes to the GPU Partitioner the used/free MIG resources of all the GPUs of the node
on which it is running through the following node annotations:

* `nos.nebuly.ai/status-gpu-<index>-<mig-profile>-free: <quantity>`
* `nos.nebuly.ai/status-gpu-<index>-<mig-profile>-used: <quantity>`

The MIG Agent also watches the node's annotations and, every time there desired MIG partitioning specified by the
GPU Partitioner does not match the current state, it tries to apply it by creating and deleting the MIG profiles
on the target GPUs. The GPU Partitioner specifies the desired MIG geometry of the GPUs of a node through annotations in
the following format:

`nos.nebuly.ai/spec-gpu-<index>-<mig-profile>: <quantity>`


Note that in some cases the MIG Agent might not be able to apply the desired MIG geometry specified by the 
GPU Partitioner. This can happen for two reasons:
1. the MIG Agent never deletes MIG resources being in use by a Pod
2. some MIG geometries require the MIG profiles to be created in a certain order, and due to reason (1) the MIG Agent 
   might not be able to delete and re-create the existing MIG profiles in the order required by the new MIG geometry. 
 
In these cases, the MIG Agent tries to apply the desired partitioning by creating as many required resources as 
possible, in order to maximize the number of schedulable Pods. This can result in the MIG Agent applying the 
desired MIG geometry only partially.

For further information regarding NVIDIA MIG and its integration with Kubernetes, please refer to the
[NVIDIA MIG User Guide](https://docs.nvidia.com/datacenter/tesla/pdf/NVIDIA_MIG_User_Guide.pdf) and to the
[MIG Support in Kubernetes](https://docs.nvidia.com/datacenter/cloud-native/kubernetes/mig-k8s.html)
official documentation provided by NVIDIA.

## MPS Partitioning


## Configuration

You can customize the GPU Partitioning settings by editing the values file of the 
[nos-gpu-partitioner](../helm-charts/nos-gpu-partitioner/README.md) Helm chart. 
In this section we focus on some of the values that you would typically want to customize.

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

The GPU Partitioner reads the scheduler configuration from the ConfigMap defined by the field 
`nos-gpu-partitioner.scheduler.config`, and it falls back to the default configuration if the ConfigMap is not found. 
You can edit this field to provide your custom scheduler configuration.

If you installed `nos` with the `nos-scheduler` flag enabled, the GPU Partitioner will use its configuration unless 
you specify a custom ConfigMap.


### Available MIG geometries
The GPU Partitioner determines the most proper partitioning plan to apply by considering the possible MIG geometries 
allowed by the GPUs of the cluster.

You can set the MIG geometries supported by each GPU model by editing the `nos-gpu-partitioner.knownMigGeometries` value 
of the [nos-gpu-partitioner Helm chart](../helm-charts/nos-gpu-partitioner/README.md). 

You can edit this file to add new MIG geometries for new GPU models, or to edit the existing ones according 
to your specific needs. For instance, you can remove some MIG geometries if you don't want to allow them to be used for a 
certain GPU model.

## Troubleshooting
If you run into issues with Automatic GPU Partitioning, you can troubleshoot by checking the logs of the GPU Partitioner
and MIG Agent pods. You can do that by running the following commands:

Check GPU Partitioner logs
```shell
 kubectl logs -l app.kubernetes.io/component=gpu-partitioner -f
```

Check MIG Agent logs:
```shell
 kubectl logs -l app.kubernetes.io/component=mig-agent -f
```
