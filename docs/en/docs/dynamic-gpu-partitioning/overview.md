# Overview

`nos` allows you to schedule Pods requesting fractions of GPUs without having to manually partition them:
the partitioning is performed automatically in real-time based on the pending and running Pods in your cluster, so that the GPUs
are always fully utilized.

With `nos`, there is no need to manually create and manage MIG configurations. 
Simply submit your Pods to the cluster and the requested MIG devices are automatically provisioned.

`nos` constantly watches the pending Pods and finds the best possible GPU partitioning configuration
to schedule the highest number of the ones requesting fractions of GPUs, which otherwise would not
be possible to schedule due to lack of resources.

You can think of it as a [Cluster Autoscaler](https://github.com/kubernetes/autoscaler) for GPUs: instead of scaling 
up the number of nodes and GPUs, it dynamically partitions them to maximize their utilization, leading to spare 
GPU capacity. Then, you can schedule more Pods or reduce the number of GPU nodes needed, reducing infrastructure costs.

The GPU partitioning is performed either using
[Multi-instance GPU (MIG)](partitioning-modes-comparison.md#multi-instance-gpu-mig) or
[Multi-Process Service (MPS)](partitioning-modes-comparison.md#multi-process-service-mps), depending on the partitioning mode
you choose for each node. 
