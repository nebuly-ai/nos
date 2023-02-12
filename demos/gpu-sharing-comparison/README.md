# MPS/Time-slicing performance comparison

The goal of this demo is compare the performance of multiple workload running on the same GPU 
shared either using Multi-Process Service (MPS) or time-slicing. 

### Time-slicing
Time-slicing consists of oversubscribing a GPU leveraging its time-slicing scheduler, which executes multiple CUDA 
processes concurrently through temporal sharing. 

The GPU, therefore, shares its compute resources among the different 
processes in a fair-sharing manner by switching between them at regular intervals of time. 

This incurs the cost of context-switching overhead, which translates into jitter and higher latency that affect the workloads.

### Multi-Process Service (MPS)
Multi-Process Service (MPS) is a client-server implementation of the CUDA Application Programming Interface (API) for 
running multiple processes concurrently on the same GPU.

The server manages GPU access providing concurrency between clients, while clients connect to it through the client 
runtime, which is built into the CUDA Driver library and may be used transparently by any CUDA application.


## Prerequisites

* Kubernetes v1.24 (required by Seldon Core)

## Steps

### 1. Install the required components
```bash
make install
```

This installs:
- NVIDIA GPU Operator
- Cert Manager
- Nos
- Nebuly NVIDIA Device Plugin
- Istio
- Seldon
- Kube Prometheus