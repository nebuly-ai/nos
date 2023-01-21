# Partitioning modes comparison

The following tables summarizes the difference between the different partitioning modes supported by NVIDIA GPUs.
Note that they are not mutually exclusive: `nos` allows you to choose a different partitioning mode for each node in your
cluster according to your needs and available hardware.

| Partitioning mode          | Supported by `nos` | Workload isolation level | Pros                                                                                                                        | Cons                                                                                                                                                        |
|----------------------------|:-------------------|:-------------------------|-----------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Multi-instance GPU (MIG)   | ✅                  | Best                     | <ul><li>Processes are executed in parallel</li><li>Full isolation (dedicated memory and compute resources)</li></ul>        | <ul><li>Supported by fewer GPU models (only Ampere or more recent architectures)</li><li>Coarse-grained control over memory and compute resources</li></ul> |
| Multi-process server (MPS) | ✅                  | Good                     | <ul><li>Processes are executed parallel</li><li>Fine-grained control over memory and compute resources allocation</li></ul> | <ul><li>No error isolation and memory protection</li></ul>                                                                                                  |
| Time-slicing               | ❌                  | None                     | <ul><li>Processes are executed concurrently</li><li>Supported by older GPU architectures (Pascal or newer)</li></ul>        | <ul><li>No resource limits</li><li>No memory isolation</li><li>Lower performance due to context-switching overhead</li></ul>                                |

## Multi-instance GPU (MIG)

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

## Multi-Process Service (MPS)

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

Additionally, MPS eliminates the context-switching overhead by executing processes in parallel through 
*spatial sharing*, resulting in higher workloads performance.

It is however important to point out that, even though allocatable memory and compute resources limits are enforced,
processes sharing a GPU through MPS are not fully isolated from each other. For instance, MPS does not provide error
isolation and memory protection, which means that a process can crash and cause the entire GPU to be reset (this
can however often been avoided by gracefully handling CUDA errors and SIGTERM signals).

## Time-slicing

Time-slicing consists of oversubscribing a GPU leveraging its time-slicing scheduler, which executes multiple CUDA
processes concurrently through *temporal sharing*. This means that the GPU shares its compute resources among the
different processes in a fair-sharing manner by switching between them at regular intervals of time. This brings
the cost of context-switching overhead, which translates into jitter and higher latency that affects the workloads.

Time-slicing also does not provide any level of memory isolation between the different processes sharing a GPU, nor
any memory allocation limits, which can lead to frequent out-of-memory (OOM) errors.

Given the drawbacks above the availability of more robust technologies such as MIG and MPS, at the moment we
decided to not support time-slicing partitioning in `nos`.
