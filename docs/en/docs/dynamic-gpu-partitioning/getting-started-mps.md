# Getting started with MPS partitioning

!!! warning
    [Multi-Process Service (MPS)](https://docs.nvidia.com/deploy/mps/index.html) is supported only by NVIDIA GPUs
    based on Volta and newer architectures.

## Prerequisites

- you need the Nebuly [k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin#installation) installed on your cluster

## Enable automatic partitioning

You can enable automatic MPS partitioning on a node by adding to it the following label:

```shell
kubectl label nodes <node-name> "nos.nebuly.ai/gpu-partitioning=mps"
```

The label delegates to `nos` the management of the MPS resources of all the GPUs of that node. You just have
to create submit your Pods to the cluster and  the requested MPS resources are automatically provisioned.

## Create pods requesting MPS resources

You can make your pods request slices of GPU by specifying MPS resources in their containers requests.
MPS devices are exposed by our k8s-device-plugin using the following naming convention:
`nvidia.com/gpu-<size>gb`, where `<size>` corresponds to the GB of memory of the GPU slice.
The computing resources are instead equally shared among all its MPS resources.

You can specify any size you want, but you should keep in mind that the GPU Partitioner will create an MPS resource
on a certain GPU only if its size is smaller or equal than the total amount of memory of that GPU (which is indicated by the
node label `nvidia.com/gpu.memory` applied by the NVIDIA GPU Operator).

For instance, you can create a pod requesting a slice of a 10GB of GPU memory as follows:

```yaml
$ kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: mps-partitioning-example
spec:
  hostIPC: true # (2)
  securityContext:
    runAsUser: 1000 # (3)
  containers:
    - name: sleepy
      image: "busybox:latest"
      command: ["sleep", "120"]
      resources:
        limits:
          nvidia.com/gpu-10gb: 1 # (1)
EOF
```

1. Fraction of GPU with 10 GB of memory
2. `hostIPC` must be set to true
3. Containers must run as the same user as the MPS Server

Pods requesting MPS resources must meet two requirements:

1. `hostIPC` must be set to `true` in order to allow containers to access the IPC namespace of the host
2. Containers must run as the same user as the user running the MPS server on the host, which is `1000` by default

The two requirements above are due to how MPS works. Since it requires the clients and the server to share the same
memory space, we need to allow the pods to access the host IPC namespace so that it can communicate with the MPS server
running on it. Moreover, the MPS server accepts only connections from clients running as the same user as the server,
which is `1000` by default (you can change it by setting the `mps.userID` value when installing the
[k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin#installation) chart), so the containers of your pods
must run with the same user if they request MPS resources.

!!! note
    Containers are supposed to request at most one MPS device. If a container needs more resources,
    then it should ask for a larger, single device as opposed to multiple smaller devices

!!! warning
    If you run `nvidia-smi` inside a container, the output still shows the whole memory of the GPU.
    Nevertheless, processes inside the container are able to allocate only the amount of memory requested by the contaner.
    You can check the availble GPU memory through the environment variable `CUDA_MPS_PINNED_DEVICE_MEM_LIMIT`.
