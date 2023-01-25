# Partitioning modes comparison

The following tables summarizes the difference between the different partitioning modes supported by NVIDIA GPUs.
Note that they are not mutually exclusive: `nos` allows you to choose a different partitioning mode for each node in your
cluster according to your needs and available hardware.

| Partitioning mode          | Supported by `nos` | Workload isolation level | Pros                                                                                                                        | Cons                                                                                                                                                        |
|----------------------------|:-------------------|:-------------------------|-----------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Multi-instance GPU (MIG)   | ✅                  | Best                     | <ul><li>Processes are executed in parallel</li><li>Full isolation (dedicated memory and compute resources)</li></ul>        | <ul><li>Supported by fewer GPU models (only Ampere or more recent architectures)</li><li>Coarse-grained control over memory and compute resources</li></ul> |
| Multi-process server (MPS) | ✅                  | Medium                     | <ul><li>Processes are executed parallel</li><li>Fine-grained control over memory and compute resources allocation</li></ul> | <ul><li>No error isolation and memory protection</li></ul>                                                                                                  |
| Time-slicing               | ❌                  | None                     | <ul><li>Processes are executed concurrently</li><li>Supported by older GPU architectures (Pascal or newer)</li></ul>        | <ul><li>No resource limits</li><li>No memory isolation</li><li>Lower performance due to context-switching overhead</li></ul>                                |

## Multi-instance GPU (MIG)

Multi-instance GPU (MIG) is a technology available on NVIDIA Ampere or more recent architectures that allows to securely
partition a GPU into separate GPU instances for CUDA applications, each fully isolated with its own high-bandwidth
memory, cache, and compute cores.

The isolated GPU slices are called MIG devices, and they are named adopting a format that indicates the compute and
memory resources of the device. For example, 2g.20gb corresponds to a GPU slice with 20 GB of memory.

MIG does not allow to create GPU slices of custom sizes and quantity, as each GPU model only supports a
[specific set of MIG profiles](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/#supported-profiles).
This reduces the degree of granularity with which you can partition the GPUs.
Additionally, the MIG devices must be created respecting certain placement rules, which further limits flexibility of use.

MIG is the GPU sharing approach that offers the highest level of isolation among processes.
However, it lacks in flexibility and it is compatible only with few GPU architectures (Ampere and Hopper).

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

Compared to time-slicing, MPS eliminates the overhead of context-switching by running processes in parallel
through spatial sharing, and therefore leads to better compute performance. Moreover, MPS provides each
client with its own GPU memory address space. This allows to enforce memory limits on the processes overcoming
the limitations of time-slicing sharing.

It is however important to point out that processes sharing a GPU through MPS are not fully isolated from each other.
Indeed, even though MPS allows to limit clients' compute and memory resources, it does not provide error isolation and
memory protection. This means that a client process can crash and cause the entire GPU to reset,
impacting all other processes running on the GPU. However, this issue can often be addressed by properly handling CUDA
errors and SIGTERM signals.

## Time-slicing

Time-slicing consists of oversubscribing a GPU leveraging its time-slicing scheduler, which executes multiple CUDA
processes concurrently through *temporal sharing*.

This means that the GPU shares its compute resources among the different processes in a fair-sharing manner
by switching between processes at regular intervals of time. This generates a computing time overhead related to
the continuous context switching, which translates into jitter and higher latency.

Time-slicing is supported by basically every GPU architecture and is the simplest solution for sharing a GPU in
a Kubernetes cluster. However, constant switching among processes creates a computation time overhead.
Also, time-slicing does not provide any level of memory isolation among the processes sharing a GPU, nor any memory
allocation limits, which can lead to frequent Out-Of-Memory (OOM) errors.

!!! info
    Given the drawbacks above the availability of more robust technologies such as MIG and MPS, at the moment we
    decided to not support time-slicing GPU sharing in `nos`.
