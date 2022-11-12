# Nebulnetes 

## Overview

### High-level features

#### Automatic MIG GPU partitioning
The GPU Partitioner watches pending pods that cannot be scheduled due to lacking GPU resources and, whenever it is
possible, it updates the MIG geometry of the MIG-enabled GPUs of the cluster in order to maximize the number of pods 
that can be scheduled.

In this way you don't have to worry about MIG partitioning anymore, you just submit your pods and Nebulnetes 
automatically takes care of finding and applying the most proper MIG geometry for providing the required resources.

#### Elastic resource quota management
Nebulnetes extends the Kubernetes [Resource Quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/)
by making them more flexible through two custom resources: `ElasticQuotas`  and `CompositeElasticQuotas`.
While standard Kubernetes resource quotas allow you only to define limits on the maximum 
overall resource allocation of each namespace, Nebulnetes elastic quota let you define two 
different limits: 
1. `min`: the minimum resources that are guaranteed to the namespace
2. `max`: the upper bound of the resources that the namespace can consume 
 
In this way namespaces can borrow reserved resource quotas from other namespaces that are not using them, 
as long as they do not exceed their max limit (if any) and the namespaces lending the quotas do not need them. 
When a namespace claims back its reserved `min` resources, pods borrowing resources from other namespaces (e.g. 
over-quota pods) can be preempted to make up space.


## Getting started with Elastic resource quotas

### Prerequisites
* it is recommended to have [cert-manager](https://cert-manager.io/docs/installation/) installed on your cluster in 
order to automatically manage the SSL certificates of the HTTP endpoints of the webhook used for validating the 
custom resources. Alternatively, you can manually create these certificates and inject them in the n8s operator 
controller manager.

### Installation

### Create elastic quotas

## Getting started with MIG Partitioner

### Prerequisites
* you need the NVIDIA GPU Operator deployed on your cluster, configured to use the "mixed" MIG strategy, 
as described in the prerequisite section
* MIG partitioning is allowed only for GPUs based on the NVIDIA Ampere and more recent architectures
(such as NVIDIA A100, NVIDIA A30, NVIDIA H100)
* if a node has multiple GPUs, all the GPUs must be of the same model

For further information regarding NVIDIA MIG partitioning and its integration in Kubernetes, please refer to the 
[NVIDIA MIG User Guide](https://docs.nvidia.com/datacenter/tesla/pdf/NVIDIA_MIG_User_Guide.pdf) and to the
[MIG Support in Kubernetes](https://docs.nvidia.com/datacenter/cloud-native/kubernetes/mig-k8s.html) 
official documentation provided by NVIDIA.

### Enable nodes for automatic partitioning
You can make a node eligible for automatic MIG partitioning by following the two steps 
described below.

#### 1. Enable MIG on the GPUs of the node
SSH to the node and run the following command for each GPU for which you want to enable MIG, 
where `<index>` correspond to the index of the GPU you want to enable:
```shell
sudo nvidia-smi -i <index> -mig 1
```
Depending on the kind of machine you are using, it may be necessary to reboot the node.

#### 2. Enable automatic MIG partitioning
Add the following label to the node in order to let Nebulnetes automatically change the MIG geometry of its GPUs:
```shell
n8s.nebuly.ai/auto-mig-enabled: "true"
```

After enabling one or more nodes for automatic MIG partitioning all you have to do is just to submits your Pods 
requesting MIG profile as resources, and Nebulnetes will take care of all the rest.

## Where to go from here

### Documentation
* [MIG GPU Partitioner](doc/mig-gpu-partitioner.md)
* [Elastic resource quota](doc/elastic-quota.md)

### Developer
* [Overview](doc/developer/overview.md)
* [Contributing to Nebulnetes]()
* [Roadmap]()

