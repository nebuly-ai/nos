# Nebulnetes 

## Overview

### High-level features

#### Automatic MIG GPU partitioning
The GPU Partitioner watches pending pods that cannot be scheduled due to lacking GPU resources and, whenever it is
possible, it updates the MIG geometry of the MIG-enabled GPUs of the cluster in order to make it possible to schedule
these pods.

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
You can make a node eligible for automatic MIG partitioning by following the steps described below.

Please note that:
* you need the NVIDIA GPU Operator deployed on your cluster and using the "mixed" MIG strategy, 
as described in the prerequisite section
* MIG partitioning is allowed only for GPUs based on the NVIDIA Ampere and most recent architectures
(such as NVIDIA A100, NVIDIA A30, NVIDIA H100)
* if a node has multiple GPUs, all the GPUs must be of the same model

For further information regarding NVIDIA MIG partitioning and its integration in Kubernetes, please refer to the 
[NVIDIA MIG User Guide](https://docs.nvidia.com/datacenter/tesla/pdf/NVIDIA_MIG_User_Guide.pdf) and to the
[MIG Support in Kubernetes](https://docs.nvidia.com/datacenter/cloud-native/kubernetes/mig-k8s.html) 
official documentation provided by NVIDIA.

### Enable MIG on the GPUs of the node