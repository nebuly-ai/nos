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
The Nebulnetes capacity-scheduling scheduler plugin extends


## Getting started

### Install n8s scheduler
Prerequisites:
* [cert-manager](https://cert-manager.io/docs/installation/): we recommend using cert-manager for provisioning the 
certificates for the webhook server used for validating the Nebulnetes custom resources.

### Install GPU partitioner
Prerequisites: 
* [nvidia-gpu-operator](https://github.com/NVIDIA/gpu-operator): mig-strategy must be set to "mixed"

## MIG GPU Partitioning

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

### How to use it
You can make a node eligible for automatic MIG partitioning by following the steps described below.

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


### How it works
Nebulnetes deploys the **MIG Agent** component on every node labelled with `n8s.nebuly.ai/auto-mig-enabled: "true"`. 
The MIG Agent exposes to Kubernetes the MIG geometry of all the GPUs of the node on which it is deployed by using 
the following Node annotations:
* `n8s.nebuly.ai/status-gpu-<index>-<mig-profile>-free: <quantity>`
* `n8s.nebuly.ai/status-gpu-<index>-<mig-profile>-used: <quantity>`

These annotations are used by the **MIG GPU Partitioner** component for reconstructing the current state of the cluster
and deciding how to change the MIG geometry of its GPUs. It does that everytime there is a batch of pending Pods that 
cannot be scheduled due to lack of MIG resources: it processes the batch and tries to update the MIG 
geometry the available GPUs by deleting unused profiles and creating the ones required by the pending pods. If the 
updated geometry allows to schedule one or more pods, the GPU Partitioner applies it by updating the spec annotations 
of the involved nodes. These annotations have the following format: 
`n8s.nebuly.ai/spec-gpu-<index>-<mig-profile>: <quantity>`

The MIG Agent watches node's annotations and everytime there desired MIG partitioning (specified with the spc annotations
mentioned above) does not match the current state, it tries to apply it by creating and deleting the MIG profiles 
on the target GPUs. 

Note that MIG profiles being used by some pod are never deleted: before applying the desired status the MIG agent 
always checks whether the profiles to delete are currently allocated.

