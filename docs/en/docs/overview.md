# Overview

`nos` is the open-source module for running AI workloads on Kubernetes in an optimized way,
increasing GPU utilization, cutting down infrastructure costs and improving workloads performance.

Currently, the available features are:

* [Dynamic GPU partitioning](docs/en/docs/dynamic-gpu-partitioning.md): `nos` ensures that each Pod uses the GPU resources
that are strictly necessary by allowing to schedule Pods requesting fractions of GPUs. GPU partitioning is performed
automatically in real-time based on the Pods pending and running in the cluster, so that GPUs are always fully utilized.
* [Elastic Resource Quota management](docs/en/docs/elastic-quota.md): increases the number of Pods running on the
cluster by allowing namespaces to borrow quotas of reserved resources from other namespaces as long as they are
not using them.

![](img/gpu-utilization.png)
