---
hide:

- toc

---

# Prerequisites

1. Kubernetes version 1.23 or newer
2. [GPU Support must be enabled](#enable-gpu-support)
3. [Nebuly's device plugin](#install-nebulys-device-plugin) (required only if using MPS partitioning)
4. [Cert Manager](https://github.com/cert-manager/cert-manager) (optional, but recommended)

## Enable GPU support

Before installing `nos`, you must enable GPU support in your Kubernetes cluster.

There are two ways to do this. One option is to manually install the required components individually,
while the other consists in installing only the NVIDIA GPU Operator, which automatically installs
all the necessary components for you. See below for more information on these two installation methods.

We recommended enabling GPU support using the NVIDIA GPU Operator (option 1).

### Option 1 - NVIDIA GPU Operator

You can install the [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) as follows:

```bash
helm install --wait --generate-name \
     -n gpu-operator --create-namespace \
     nvidia/gpu-operator --version v22.9.0 \
     --set driver.enabled=true \ 
     --set migManager.enabled=false \ 
     --set mig.strategy=mixed \ 
     --set toolkit.enabled=true
```

Note that the GPU Operator will automatically install a recent version of NVIDIA Drivers and CUDA on all the GPU-enabled
nodes of your cluster, so you don't have to manually install them.

For further information you can refer to the
[NVIDIA GPU Operator Documentation](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/getting-started.html).

### Option 2 - Manual installation

!!! warning

    If you want to enable MPS Dynamic Partitioning, make sure you have a version of CUDA 11.5 or newer 
    installed, as this is the minimum version that supports GPU memory limits in MPS.

To enble GPU support in your cluster, you first need to install
[NVIDIA Drivers](https://www.nvidia.com/download/index.aspx) and the
[NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html)
on all the nodes of your cluster with a GPU.

After installing the NVIDIA Drivers and the Container Toolkit on your nodes, you need to install the
following Kubernetes components:

- [NVIDIA GPU Feature Discovery](https://github.com/NVIDIA/gpu-feature-discovery)
- [NVIDIA Device Plugin](https://github.com/NVIDIA/k8s-device-plugin)

Please note that the configuration parameter `migStrategy` must be set to `mixed` (you can do that
with `--set migStrategy=mixed`
if you are using Helm).

## Install Nebuly's device plugin

!!! info

    Nebuly's device plugin is required only if you want to use [dynamic MPS partitioning](#dynamic-gpu-partitioning/getting-started-mps.md).
    If you don't plan to use MPS partitioning, you can then skip this installation step.

You can install [Nebuly's device plugin](https://github.com/nebuly-ai/k8s-device-plugin) using Helm as follows:

```bash
helm install oci://ghcr.io/nebuly-ai/helm-charts/nvidia-device-plugin \
  --version 0.13.0 \
  --generate-name \
  -n nebuly-nvidia \
  --create-namespace
```

Nebuly's device plugin runs only on nodes labelled with `nos.nebuly.com/gpu-partitioning=mps`.

If you already have the NVIDIA device plugin installed on your cluster, you need to ensure that only
one instance of the device plugin is running on each GPU node (either Nebuly's or NVIDIA's).
One way to do that is to add an affinity rule to the NVIDIA device plugin Daemonset so that it doesn't
run on any node that has MPS enabled:

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: nos.nebuly.com/gpu-partitioning
              operator: NotIn
              values:
                - mps
```

For further information you can refer to
[Nebuly's device plugin documentation](https://github.com/nebuly-ai/k8s-device-plugin#installation-alongside-the-nvidia-device-plugin).
