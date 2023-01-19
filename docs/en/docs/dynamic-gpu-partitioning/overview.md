# Overview

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
