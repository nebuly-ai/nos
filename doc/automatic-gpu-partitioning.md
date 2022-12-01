# Automatic GPU partitioning

> ⚠️ At the moment Nebulnetes only
> supports [Multi-instance GPU (MIG) partitioning](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/index.html),
> which is available only for NVIDIA GPUs based on Ampere and Hopper architectures.

## Table of contents
- [Overview](#overview)
- [Getting started](#getting-started)
- [Enable nodes for automatic partitioning](#enable-nodes-for-automatic-partitioning)
- [MIG partitioning](#mig-partitioning)
- [Configuration](#configuration)
  - [Pods batch size](#pods-batch-size)
  - [Scheduler configuration](#scheduler-configuration)
  - [Integration with Nebulnetes scheduler](#integration-with-nebulnetes-scheduler)
  - [Available MIG geometries](#available-mig-geometries)
- [Troubleshooting](#troubleshooting)
- [Uninstall](#uninstall)

## Overview

Nebulnetes allows you to schedule Pods requesting fractions of GPUs without having to manually partition them:
the partitioning is performed dynamically based on the pending and running Pods in your cluster, so that the GPUs
are always fully utilized.

The GPU partitioning is performed by the [GPU Partitioner](../config/gpupartitioner) component, which
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

## Getting started 

### Prerequisites

* you need the [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) deployed on your cluster, configured to
  use the `mixed` MIG strategy
* you need at least one node with a GPU supporting [MIG](https://www.nvidia.com/en-us/technologies/multi-instance-gpu/)
* if a node has multiple GPUs, all the GPUs must be of the same model

For further information regarding NVIDIA MIG and its integration with Kubernetes, please refer to the
[NVIDIA MIG User Guide](https://docs.nvidia.com/datacenter/tesla/pdf/NVIDIA_MIG_User_Guide.pdf) and to the
[MIG Support in Kubernetes](https://docs.nvidia.com/datacenter/cloud-native/kubernetes/mig-k8s.html)
official documentation provided by NVIDIA.

### Installation
You can install the automatic GPU partitioning components by running the Makefile targets below, which deploys them
required to the k8s cluster specified in your `~/.kube/config`.

By default, all the resources are installed in the `n8s-system` namespace.

1. Deploy the Nebulnetes operator

```shell
make deploy-operator
```

2. Deploy the GPU Partitioner

```shell
make deploy-gpu-partitioner
```

3. Deploy the MIG Agent

```shell
make deploy-mig-agent
```

The targets above deploy the components using their default configuration. If you want to customize their configuration,
you can refer to the [GPU Partitioner Configuration](doc/automatic-gpu-partitioning.md#configuration) page for more
information.

## How to enable automatic GPU partitioning on a Node

> ⚠️ Prerequisite: to enable automatic MIG partitioning on a node, first you need to enable MIG mode on its GPUs.
You can do that by running the following command for each GPU want to enable,
where `<index>` correspond to the index of the GPU: `sudo nvidia-smi -i <index> -mig 1`
> 
> Depending on the kind of machine you are using, it may be necessary to reboot the node after enabling MIG mode for
> one of its GPUs.


You can enable automatic MIG partitioning on a node by adding to it the following label:

```shell
kubectl label nodes <node-name> "n8s.nebuly.ai/gpu-partitioning=mig"
```

The label delegates to Nebulnetes the management of the MIG resources of all the GPUs of that node, so you don't have 
to manually configure the MIG geometry of the GPUs anymore: n8s will dynamically apply the most proper geometry 
according to the resources requested by the pods submitted to the cluster.


## MIG partitioning
In the case of MIG partitioning, the agent that creates/deletes the MIG resources is 
the [MIG Agent](../config/migagent), which is a daemonset running on every node labeled 
with `n8s.nebuly.ai/gpu-partitioning: mig`.

The MIG Agent exposes to the GPU Partitioner the used/free MIG resources of all the GPUs of the node
on which it is running through the following node annotations:

* `n8s.nebuly.ai/status-gpu-<index>-<mig-profile>-free: <quantity>`
* `n8s.nebuly.ai/status-gpu-<index>-<mig-profile>-used: <quantity>`

The MIG Agent also watches the node's annotations and, every time there desired MIG partitioning specified by the
GPU Partitioner does not match the current state, it tries to apply it by creating and deleting the MIG profiles
on the target GPUs. The GPU Partitioner specifies the desired MIG geometry of the GPUs of a node through annotations in
the following format:

`n8s.nebuly.ai/spec-gpu-<index>-<mig-profile>: <quantity>`


Note that in some cases the MIG Agent might not be able to apply the desired MIG geometry specified by the 
GPU Partitioner. This can happen for two reasons:
1. the MIG Agent never deletes MIG resources being in use by a Pod
2. some MIG geometries require the MIG profiles to be created in a certain order, and due to reason (1) the MIG Agent 
   might not be able to delete and re-create the existing MIG profiles in the order required by the new MIG geometry. 
 
In these cases, the MIG Agent tries to apply the desired partitioning by creating as many required resources as 
possible, in order to maximize the number of schedulable Pods. This can result in the MIG Agent applying the 
desired MIG geometry only partially.

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

### Integration with Nebulnetes scheduler

If you want to use Automatic GPU partitioning together with the Nebulnetes scheduler so that Elastic Resource quotas
are taken into account when performing the GPUs partitioning, you can follow these steps:

* deploy the Nebulnetes scheduler
* uncomment the last patch of this [kustomization file](../config/gpupartitioner/default/kustomization.yaml), which
  mounts the n8s scheduler config file to the GPU partitioner pod filesystem
* set the `schedulerConfigFile` value to `scheduler_config.yam`

### Available MIG geometries
The GPU Partitioner determines the most proper partitioning plan to apply by considering the possible MIG geometries 
allowed by the GPUs of the cluster.

The MIG geometries allowed by each known GPU model are specified in the configuration file 
[known_mig_geometries.yaml](../config/gpupartitioner/manager/known_mig_geometries.yaml), 
which is provided to the GPU partitioner through the configuration param `knownMigGeometriesFile`.

You can edit this file to add new MIG geometries for new GPU models, or to edit the existing ones according 
to your specific needs. For instance, you can remove some MIG geometries if you don't want to allow them to be used for a 
certain GPU model.

## Troubleshooting
If you run into issues with Automatic GPU Partitioning, you can troubleshoot by checking the logs of the GPU Partitioner
and MIG Agent pods. You can do that by running the following commands:

Check GPU Partitioner logs
```shell
 kubectl logs -n n8s-system -l app.kubernetes.io/component=gpu-partitioner -f
```

Check MIG Agent logs:
```shell
 kubectl logs -n n8s-system -l app.kubernetes.io/component=mig-agent -f
```

### How to increase log verbosity
You can increase the log verbosity by providing the argument `--zap-log-level=<level>` to the 
GPU Partitioner and MIG Agent containers, where `<level>` is an integer between 0 and 3. 
Higher values means higher verbosity (0 is the default value, 1 corresponds to the DEBUG level).

You can do that by editing the [MIG Agent](../config/migagent/default/mig_agent_config_patch.yaml) and 
[GPU Partitioner](../config/gpupartitioner/default/gpu_partitioner_config_patch.yaml) Kustomize manifests and 
re-deploying the two components.

## Uninstall
To uninstall Automatic GPU Partitioning, you can run the following command:

```shell
make undeploy-mig-agent ; make undeploy-gpu-partitioner
```