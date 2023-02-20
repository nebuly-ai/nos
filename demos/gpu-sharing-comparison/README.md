# GPU Sharing performance comparison

In this demo we compare different GPU-sharing technologies in terms of how they affect the performance
of the workloads running on the same GPU. In particular, we address the following technologies:

* Multi-Process Service (MPS)
* Multi-Instance GPU (MIG)
* Time-slicing

We measure the offered by each GPU-sharing techniques by running a set of Pods on the same shared GPU.
Each Pod has a simple container that constantly runs
inferences on a [YOLOS](https://huggingface.co/hustvl/yolos-small) model with a sample input image. The execution time
of each inference is collected and exported by Prometheus, so that later it can be easily queried and visualized.

We execute this experiment multiple times, each time with a different number of Pods running on the same GPU
(1, 3, 5 and 7), and we repeat this processes for each GPU-sharing technology.

## Table of Contents
* [GPU Sharing technologies overview](#gpu-sharing-technologies-overview)
* [Experiments](#experiments)
* [Results](#results)
* [How to run experiments](#how-to-reproduce)

## GPU Sharing technologies overview

### Time-slicing

[Time-slicing](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/gpu-sharing.html)
consists of oversubscribing a GPU leveraging its time-slicing scheduler, which executes multiple CUDA processes 
concurrently through temporal sharing.

The GPU, therefore, shares its compute resources among the different
processes in a fair-sharing manner by switching between them at regular intervals of time.

This incurs the cost of context-switching overhead, which translates into jitter and higher latency that affect the
workloads.

### Multi-Process Service (MPS)

[Multi-Process Service (MPS)](https://docs.nvidia.com/deploy/mps/index.html) is a client-server implementation of the 
CUDA Application Programming Interface (API) for running multiple processes concurrently on the same GPU.

The server manages GPU access providing concurrency between clients, while clients connect to it through the client
runtime, which is built into the CUDA Driver library and may be used transparently by any CUDA application.

### Multi-Instance GPU (MIG)
[Multi-Instance GPU (MIG)](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/) is a technology available on 
NVIDIA Ampere and Hopper architectures that allows to securely partition a GPU into up to seven separate GPU instances, 
each fully isolated with its own high-bandwidth memory, cache, and compute cores.

MIG is the GPU sharing approach that offers the highest level of isolation among processes. 
However, it lacks flexibility and it is compatible only with few GPU architectures (Ampere and Hopper).

## Experiments

## Results

|              | 7 Pods              | 5 Pods             | 3 Pods             | 1 Pod               |
|--------------|---------------------|--------------------|--------------------|---------------------|
| Time-slicing | 0.6848803898344092  | 0.4889516484510863 | 0.2931403323839484 | 0.08815048247208879 |
| MPS          | 0.3198182253208076  | 0.2408641491144998 | 0.1640018804617158 | 0.08796154366177533 |
| MIG          | 0.34424962169772633 | 0.3453250103939391 | 0.3413140464128765 | 0.34236589893939284 |

## How to run experiments

### 1. Install the required components

```bash
make install
```

This installs:

- NVIDIA GPU Operator
- Cert Manager
- Nos
- Nebuly NVIDIA Device Plugin
- Kube Prometheus