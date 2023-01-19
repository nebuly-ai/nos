# Configuration

You can customize the GPU Partitioner settings by editing the values file of the
[nos-gpu-partitioner](../helm-charts/nos-gpu-partitioner/README.md) Helm chart.
In this section we focus on some of the values that you would typically want to customize.

## Pods batch size

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

## Scheduler configuration

The GPU Partitioner uses an internal scheduler to simulate the scheduling of the pending pods to determine whether
a candidate GPU partitioning plan would make the pending pods schedulable.

The GPU Partitioner reads the scheduler configuration from the ConfigMap defined by the field
`nos-gpu-partitioner.scheduler.config`, and it falls back to the default configuration if the ConfigMap is not found.
You can edit this field to provide your custom scheduler configuration.

If you installed `nos` with the `nos-scheduler` flag enabled, the GPU Partitioner will use its configuration unless
you specify a custom ConfigMap.

## Available MIG geometries

The GPU Partitioner determines the most proper partitioning plan to apply by considering the possible MIG geometries
allowed each of the GPU models present in the cluster.

You can set the MIG geometries supported by each GPU model by editing the `nos-gpu-partitioner.knownMigGeometries` value
of the [installation chart](../helm-charts/nos/README.md).

You can edit this file to add new MIG geometries for new GPU models, or to edit the existing ones according
to your specific needs. For instance, you can remove some MIG geometries if you don't want to allow them to be used for a
certain GPU model.

## How it works

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

### MIG Partitioning

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

### MPS Partitioning

The creation and deletion of MPS resources is handled by the k8s-device-plugin, which can expose a single GPU as
multiple MPS resources according to its configuration.

When allocating a container requesting an MPS resource, the device plugin takes care of injecting the
environment variables and mounting the volumes required by the container to communicate to the MPS server, making
sure that the resource limits defined by the device requested by the container are enforced.

For more information about MPS integration with Kubernetes you can refer to the
Nebuly [k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin) documentation.
