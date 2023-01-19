# Getting started with MIG partitioning

!!! warning
    [Multi-instance GPU (MIG)](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/index.html) mode
    is supported only by NVIDIA GPUs based on Ampere, Hopper and newer architectures.

## Prerequisites

- you need the [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) installed on your cluster
  - MIG strategy must be set to `mixed` (`--set mig.strategy=mixed`)
  - mig-manager must be disabled (`--set migManager.enabled=false`)
- if a node has multiple GPUs, all the GPUs must be of the same model
- all the GPUs of the nodes for which you want to enable MIG partitioning must have MIG mode enabled

## Enable MIG mode

By default, MIG is not enabled on GPUs. In order to enable it, SSH into the node and run the following command for
each GPU you want to enable MIG, where `<index>` corresponds to the index of each GPU:

```bash
sudo nvidia-smi -i <index> -mig 1
```

Depending on the kind of machine you are using, it may be necessary to reboot the node after enabling MIG mode
for one of its GPUs.

You can check whether MIG mode has been successfully enabled by running the following command and checking if you
get a similar output:

```bash
$ nvidia-smi -i <index> --query-gpu=pci.bus_id,mig.mode.current --format=csv

pci.bus_id, mig.mode.current
00000000:36:00.0, Enabled
```

For more information and troubleshooting you can refer to th<!-- e -->
[NVIDIA documentation](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/#enable-mig-mode).

## Enable automatic partitioning

You can enable automatic MIG partitioning on a node by adding to it the following label:

```shell
kubectl label nodes <node-name> "nos.nebuly.ai/gpu-partitioning=mig"
```

The label delegates to `nos` the management of the MIG resources of all the GPUs of that node, so you don't have
to manually configure the MIG geometry of the GPUs anymore: `nos` will dynamically create and delete the MIG profiles
according to the resources requested by the pods submitted to the cluster, within the limits of the possible MIG geometries
supported by each GPU model.

The available MIG geometries supported by each GPU model are defined in a ConfigMap, which by default contains
with the supported geometries of the most popular GPU models. You can override or extend the values of this
ConfigMap by editing the field `nos-gpu-partitioner.knownMigGeometries` of the
[installation chart](../helm-charts/nos/README.md).

## Create pods requesting MIG resources

You can make your pods request slices of GPU by specifying MIG devices in their containers requests:

```yaml
$ kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: mig-partitioning-example
spec:
  containers:
    - name: sleepy
      image: "busybox:latest"
      command: ["sleep", "120"]
      resources:
        limits:
          nvidia.com/mig-1g.10gb: 1
EOF
```

In the example above, the pod requests a slice of a 10GB of memory, which is the smallest unit available in
`NVIDIA-A100-80GB-PCIe` GPUs. If in your cluster you have different GPU models, the `nos` might not be able to create
the specified MIG resource. You can find the MIG profiles supported by each GPU model in the
[NVIDIA documentation](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/#supported-profiles).

Note that each container is supposed to request at most one MIG device: if a container needs more resources,
then it should ask for a larger, single device as opposed to multiple smaller devices.
