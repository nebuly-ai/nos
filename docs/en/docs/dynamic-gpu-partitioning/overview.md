# Overview

`nos` allows you to schedule Pods requesting fractions of GPUs. The GPUs are automatically
partitioned into slices that can be requested by individual containers. In this way, 
GPUs are shared among multiple Pods increasing the overall utilization.

The GPUs partitioning is performed automatically in real-time based on the requests of 
the Pods in your cluster. 
`nos` constantly watches the pending Pods and finds the best possible GPU partitioning configuration
to schedule the highest number of the ones requesting fractions of GPUs.

You can think of `nos` as a [Cluster Autoscaler](https://github.com/kubernetes/autoscaler) for GPUs: instead of 
adjusting the number of nodes and GPUs, it dynamically partitions them to maximize their utilization, leading to spare 
GPU capacity. Then, you can schedule more Pods or reduce the number of GPU nodes needed, reducing infrastructure costs.

The GPU partitioning is performed either using
[Multi-instance GPU (MIG)](partitioning-modes-comparison.md#multi-instance-gpu-mig) or
[Multi-Process Service (MPS)](partitioning-modes-comparison.md#multi-process-service-mps), depending on the partitioning mode
you choose for each node. 
