# Dynamic GPU partitioning

## Overview

`nos` allows you to schedule Pods requesting fractions of GPUs without having to manually partition them:
the partitioning is performed automatically in real-time based on the pending and running Pods in your cluster, so that the GPUs
are always fully utilized.

The [GPU Partitioner](./helm-charts/nos-gpu-partitioner/README.md) component
constantly watches the pending Pods and finds the best possible GPU partitioning configuration
to schedule the highest number of the ones requesting fractions of GPUs, which otherwise would not 
be possible to schedule due to lack of resources.

You can see it as a sort of [Cluster Autoscaler](https://github.com/kubernetes/autoscaler) for GPUs:
instead of scaling up the number of nodes and GPUs, it dynamically partitions the available GPUs to maximize
their utilization, leading to spare GPU capacity that can reduce the number of required GPU nodes (and thus the costs
of your cluster).

The GPU partitioning is performed either using
[Multi-instance GPU (MIG)](#multi-instance-gpu-mig) or
[Multi-Process Service (MPS)](#multi-process-service-mps), depending on the partitioning mode
you choose for each node. You can find more info about the partitioning modes in the section below.

* [Get started with MIG partitioning](#getting-started-with-mig-partitioning)
* [Get started with MPS partitioning](#getting-started-with-mps-partitioning)

## Partitioning modes comparison

The following tables summarizes the difference between the different partitioning modes supported by NVIDIA GPUs.
Note that they are not mutually exclusive: `nos` allows you to choose a different partitioning mode for each node in your
cluster according to your needs and available hardware.

| Partitioning mode          | Supported by `nos` | Workload isolation level | Pros                                                                                                                            | Cons                                                                                                                                                        |
|----------------------------|:-------------------|:-------------------------|---------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Multi-instance GPU (MIG)   | ✅                  | Best                     | <ul><li>Processes are executed concurrently</li><li>Full isolation (dedicated memory and compute resources)</li></ul>           | <ul><li>Supported by fewer GPU models (only Ampere or more recent architectures)</li><li>Coarse-grained control over memory and compute resources</li></ul> |
| Multi-process server (MPS) | ✅                  | Good                     | <ul><li>Processes are executed concurrently</li><li>Fine-grained control over memory and compute resources allocation</li></ul> | <ul><li>No error isolation and memory protection</li></ul>                                                                                                  |
| Time-slicing               | ❌                  | None                     | <ul><li>Supported by older GPU architectures (Pascal or newer)</li></ul>                                                        | <ul><li>No resource limits</li><li>No memory isolation</li><li>Lower performance due to context-switching overhead</li></ul>                                |

### Multi-instance GPU (MIG)

Multi-instance GPU (MIG) is a technology available on NVIDIA Ampere or more recent architectures that allows to securely
partition a GPU into separate GPU instances for CUDA applications, each fully isolated with its own high-bandwidth
memory, cache, and compute cores.

The isolated GPU slices created through MIG are called MIG devices, and they are named according to the following
naming convention: `<gpu-instance>g.<gpu-memory>gb`, where the GPU instance part corresponds to the computing
resources of the device, while the GPU Memory indicates its GB of memory. Example: `2g.20gb`

Each GPU model allows only a pre-defined set of MIG Geometries (e.g. set of MIG devices with the respective max quantity),
which limits the granularity of the partitioning. Moreover, the MIG devices allowed by a certain geometry must be
created in a specific order, further limiting the flexibility of the partitioning.

Even though MIG partitioning is less flexible, it is important to point out that it is the technology that offers the
highest level of isolation among the created GPU slices, which are equivalent to independent and fully-isolated
different GPUs.

You can find out more on how MIG technology works in the official
[NVIDIA MIG User Guide](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/).

### Multi-Process Service (MPS)

Multi-Process Service (MPS) is a client-server implementation of the CUDA Application Programming Interface (API)
for running multiple processes concurrently on the same GPU:

- the server manages GPU access providing concurrency between clients
- clients connect to the server through the client runtime, which is built into the CUDA Driver library
  and may be used transparently by any CUDA application.

The main advantage of MPS is that it provides a fine-grained control over the GPU assigned to each client, allowing to
specify arbitrary limits on both the amount of allocatable memory and the available compute. The Nebuly
[k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin)
takes advantage of this feature for exposing to Kubernetes GPU resources with an arbitrary amount of allocatable
memory defined by the user.

It is however important to point out that, even though allocatable memory and compute resources limits are enforced,
processes sharing a GPU through MPS are not fully isolated from each other. For instance, MPS does not provide error
isolation and memory protection, which means that a process can crash and cause the entire GPU to be reset (this
can however often been avoided by gracefully handling CUDA errors and SIGTERM signals).

### Time-slicing

Time-slicing consists of oversubscribing a GPU leveraging its time-slicing scheduler, which executes multiple CUDA
processes concurrently through *temporal sharing*. This means that the GPU shares its compute resources among the
different processes in a fair-sharing manner by switching between them at regular intervals of time. This brings
the cost of context-switching overhead, which translates into jitter and higher latency that affects the workloads.

Time-slicing also does not provide any level of memory isolation between the different processes sharing a GPU, nor
any memory allocation limits, which can lead to frequent out-of-memory (OOM) errors.

Given the drawbacks above the availability of more robust technologies such as MIG and MPS, at the moment we
decided to not support time-slicing partitioning in `nos`.

## Getting started with MIG partitioning

!!! warning
    [Multi-instance GPU (MIG)](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/index.html) mode
    is supported only by NVIDIA GPUs based on Ampere, Hopper and newer architectures.

### Prerequisites

- you need the [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) installed on your cluster
  - MIG strategy must be set to `mixed` (`--set mig.strategy=mixed`)
  - mig-manager must be disabled (`--set migManager.enabled=false`)
- if a node has multiple GPUs, all the GPUs must be of the same model
- all the GPUs of the nodes for which you want to enable MIG partitioning must have MIG mode enabled

### Enable MIG mode

By default, MIG is not enabled on GPUs. In order to enable it, SSH into the node and run the following command for
each GPU you want to enable MIG, where `<index>` corresponds to the index of each GPU:

```bash
sudo nvidia-smi -i <index> -mig 1
```

Depending on the kind of machine you are using, it may be necessary to reboot the node after enabling MIG mode
for one of its GPUs.

You can check whether MIG mode has been successfully enabled by running the following command and checking if you
get a similar output:

```bash
$ nvidia-smi -i <index> --query-gpu=pci.bus_id,mig.mode.current --format=csv

pci.bus_id, mig.mode.current
00000000:36:00.0, Enabled
```

For more information and troubleshooting you can refer to th<!-- e -->
[NVIDIA documentation](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/#enable-mig-mode).

### Enable automatic partitioning

You can enable automatic MIG partitioning on a node by adding to it the following label:

```shell
kubectl label nodes <node-name> "nos.nebuly.ai/gpu-partitioning=mig"
```

The label delegates to `nos` the management of the MIG resources of all the GPUs of that node, so you don't have
to manually configure the MIG geometry of the GPUs anymore: `nos` will dynamically create and delete the MIG profiles
according to the resources requested by the pods submitted to the cluster, within the limits of the possible MIG geometries
supported by each GPU model.

The available MIG geometries supported by each GPU model are defined in a ConfigMap, which by default contains
with the supported geometries of the most popular GPU models. You can override or extend the values of this
ConfigMap by editing the field `nos-gpu-partitioner.knownMigGeometries` of the
[installation chart](./helm-charts/nos/README.md).

### Create pods requesting MIG resources

You can make your pods request slices of GPU by specifying MIG devices in their containers requests:

```yaml
$ kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: mig-partitioning-example
spec:
  containers:
    - name: sleepy
      image: "busybox:latest"
      command: ["sleep", "120"]
      resources:
        limits:
          nvidia.com/mig-1g.10gb: 1
EOF
```

In the example above, the pod requests a slice of a 10GB of memory, which is the smallest unit available in
`NVIDIA-A100-80GB-PCIe` GPUs. If in your cluster you have different GPU models, the `nos` might not be able to create
the specified MIG resource. You can find the MIG profiles supported by each GPU model in the
[NVIDIA documentation](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/#supported-profiles).

Note that each container is supposed to request at most one MIG device: if a container needs more resources,
then it should ask for a larger, single device as opposed to multiple smaller devices.

## Getting started with MPS partitioning

!!! warning
    [Multi-Process Service (MPS)](https://docs.nvidia.com/deploy/mps/index.html) is supported only by NVIDIA GPUs
    based on Volta and newer architectures.

### Prerequisites

- you need the Nebuly [k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin#installation) installed on your cluster

### Enable automatic partitioning

You can enable automatic MPS partitioning on a node by adding to it the following label:

```shell
kubectl label nodes <node-name> "nos.nebuly.ai/gpu-partitioning=mps"
```

The label delegates to `nos` the management of the MPS resources of all the GPUs of that node, so you just have
to create Pods requesting MPS resources and `nos` will automatically configure the k8s-device-plugin for creating and
exposing those resources to the cluster.

### Create pods requesting MPS resources

You can make your pods request slices of GPU by specifying MPS resources in their containers requests.
MPS devices are exposed by our k8s-device-plugin using the following naming convention:
`nvidia.com/gpu-<size>gb`, where `<size>` corresponds to the GB of memory of the GPU slice.

You can specify any size you want, but you should keep in mind that the GPU Partitioner will create an MPS resource
on a certain GPU only if its size is smaller or equal than the total amount of memory of that GPU (which is indicated by the
node label `nvidia.com/gpu.memory` applied by the NVIDIA GPU Operator).

For instance, you can create a pod requesting a slice of a 10GB of GPU memory as follows:

```yaml
$ kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: mps-partitioning-example
spec:
  hostIPC: true 
  securityContext:
    runAsUser: 1000
  containers:
    - name: sleepy
      image: "busybox:latest"
      command: ["sleep", "120"]
      resources:
        limits:
          nvidia.com/gpu-10gb: 1
EOF
```

Pods requesting MPS resources must meet two requirements:

1. `hostIPC=true` is required in order to allow the container to access the IPC namespace of the host
2. the containers must run as the same user as the user running the MPS server on the host, which is `1000` by default

The two requirements above are due to how MPS works. Since it requires the clients and the server to share the same
memory space, we need to allow the pods to access the host IPC namespace so that it can communicate with the MPS server
running on it. Moreover, the MPS server accepts only connections from clients running as the same user as the server,
which is `1000` by default (you can change it by setting the `mps.userID` value when installing the
[k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin#installation) chart), so the containers of your pods
must run with the same user if they request MPS resources.

Please note that:

- as for MIG resources, each container is supposed to request at most one MPS device: if a container needs more resources,
then it should ask for a larger, single device as opposed to multiple smaller devices
- the computing resources of a GPU are equally shared among all its MPS resources
- the output of `nvidia-smi` run inside a container requesting MPS resources still shows the whole memory of the respective
GPU. Nevertheless, the container is only able to access the amount of memory of the MPS slice it requested, which is
specified by the environment variable `CUDA_MPS_PINNED_DEVICE_MEM_LIMIT`.

## Configuration

You can customize the GPU Partitioner settings by editing the values file of the
[nos-gpu-partitioner](helm-charts/nos-gpu-partitioner/README.md) Helm chart.
In this section we focus on some of the values that you would typically want to customize.

### Pods batch size

The GPU partitioner processes pending pods in batches of configurable size. You can set the batch size by editing the
following two parameters of the configuration:

- `batchWindowTimeoutSeconds`: timeout of the time window used for batching pending Pods. The time window starts
  when the GPU Partitioner starts processing a batch of pending Pods, and ends when the timeout expires or the
  batch is completed.
- `batchWindowIdleSeconds`: idle time before a batch of pods is considered completed. Once the time window of a batch
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
allowed each of the GPU models present in the cluster.

You can set the MIG geometries supported by each GPU model by editing the `nos-gpu-partitioner.knownMigGeometries` value
of the [installation chart](helm-charts/nos/README.md).

You can edit this file to add new MIG geometries for new GPU models, or to edit the existing ones according
to your specific needs. For instance, you can remove some MIG geometries if you don't want to allow them to be used for a
certain GPU model.

### How it works

The GPU Partitioner component watches for pending pods that cannot be scheduled due to lack of MIG/MPS resources
they request. If it finds such pods, it checks the current partitioning state of the GPUs in the cluster
and tries to find a new partitioning state that would allow to schedule them without deleting any of the used resources.

It does that by using an internal k8s scheduler, so that before choosing a candidate partitioning, the GPU Partitioner
simulates the scheduling to check whether the partitioning would actually allow to schedule the pending Pods. If multiple
partitioning configuration can be used to schedule the pending Pods, the one that would result in the highest number of schedulable
pods is chosen.

Moreover, just in the case of MIG partitioning, each specific GPU model allows to create only certain combinations of MIG profiles,
which are called MIG geometries, so the GPU partitioner takes this constraint into account when trying to find a
new partitioning. The available MIG geometries of each GPU model are defined in the field `nos-gpu-partitioner.knownMigGeometries` field of the Helm chart.

#### MIG Partitioning

The actual partitioning specified by the GPU Partitioner for MIG GPUs is performed by the MIG Agent, which is a daemonset running on every node labeled
with `nos.nebuly.ai/gpu-partitioning: mig` that creates/deletes MIG profiles as requested by the GPU Partitioner.

The MIG Agent exposes to the GPU Partitioner the used/free MIG resources of all the GPUs of the node
on which it is running through the following node annotations:

- `nos.nebuly.ai/status-gpu-<index>-<mig-profile>-free: <quantity>`
- `nos.nebuly.ai/status-gpu-<index>-<mig-profile>-used: <quantity>`

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

#### MPS Partitioning

The creation and deletion of MPS resources is handled by the k8s-device-plugin, which can expose a single GPU as
multiple MPS resources according to its configuration.

When allocating a container requesting an MPS resource, the device plugin takes care of injecting the
environment variables and mounting the volumes required by the container to communicate to the MPS server, making
sure that the resource limits defined by the device requested by the container are enforced.

For more information about MPS integration with Kubernetes you can refer to the
Nebuly [k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin) documentation.

## Troubleshooting

If you run into issues with Automatic GPU Partitioning, you can troubleshoot by checking the logs of the GPU Partitioner
and MIG Agent pods. You can do that by running the following commands:

Check GPU Partitioner logs:

```shell
 kubectl logs -n nebuly-nos -l app.kubernetes.io/component=nos-gpu-partitioner -f
```

Check MIG Agent logs:

```shell
 kubectl logs -n nebuly-nos -l app.kubernetes.io/component=nos-mig-agent -f
```

Check Nebuly k8s-device-plugin logs:

```shell
kubectl logs -n nebuly-nos -l app.kubernetes.io/name=nebuly-k8s-device-plugin -f
```
