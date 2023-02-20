# GPU Sharing performance comparison

In this demo we compare different GPU-sharing technologies in terms of how they affect the performance
of workloads running on the same shared GPU. We benchmark the following technologies:

* Multi-Process Service (MPS)
* Multi-Instance GPU (MIG)
* Time-slicing

We measure the performance of each GPU-sharing techniques by running a set of Pods on the same shared GPU.
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
However, it lacks flexibility, and it is compatible only with few GPU architectures (Ampere and Hopper).

## Experimental setup

We run all the experiments on an AKS cluster running Kubernetes v1.24.9 on 2 nodes:

* 1x [Standard_B2ms](https://learn.microsoft.com/en-us/azure/virtual-machines/sizes-b-series-burstable)
  (2 vCPUs, 8 GB RAM) - System node pool required by AKS.
* 1x [Standard_NC24ads_A100_v4](https://learn.microsoft.com/en-us/azure/virtual-machines/nc-a100-v4-series)
  (24 vCPUs, 220 GB RAM,
  1x [NVIDIA A100 80GB PCIe](https://www.nvidia.com/content/dam/en-zz/Solutions/Data-Center/a100/pdf/PB-10577-001_v02.pdf))

On the GPU-enabled node, we installed the following components:

* NVIDIA Container Toolkit: 1.11.0
* NVIDIA Drivers: 525.60.13

We run the benchmarks by creating a Deployment with a single Pod running the [Benchmarks Client](client)
container. We created a different deployment for each GPU sharing technology. In each deployment, 
the benchmarks container always request a single GPU slice resource:

* MIG: `nvidia.com/mig-1g.5gb: 1`
* MPS:  `nvidia.com/gpu-5gb: 1`
* Time-slicing: `nvidia.com/gpu.shared: 1`

We controlled how many containers were running on the same GPU by adjusting the number of replicas of the Deployment.


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