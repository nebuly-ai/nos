# Nebuly Operating System (nos)

`nos` is the open-source module for running AI workloads on Kubernetes in an optimized way, both in terms of
hardware utilization and workload performance.

The operating system layer is responsible for workloads scheduling and hardware abstraction.
It orchestrates the workloads taking into account considerations specific for AI/ML workloads and leveraging
techniques typical of High-performance Computing (HPC), and it hides the underlying hardware complexities.

Currently, this layer provides two features [Automatic GPU partitioning](doc/automatic-gpu-partitioning.md) and
[Elastic Resource Quota management](doc/elastic-quota.md).

## Getting started

### Prerequisites
* [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator)
* [Nebuly k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin) (optional, required only if you want to enable MPS partitioning)
* [cert-manager](https://cert-manager.io/docs/) (optional, but recommended)

### Installation
You can install `nos` using Helm 3 (recommended).
You can find all the available configuration values in the Chart [README.md](helm-charts/nos/README.md).
```bash
helm install oci://ghcr.io/nebuly-ai/helm-charts/nos \
  --version 0.1.0 \
  --namespace nebuly-nos \
  --generate-name \
  --create-namespace
```

Alternatively, you can use Kustomize by cloning the repository and running `make deploy`.

### Enable GPU partitioning
You can enable automatic GPU partitioning on a node by labelling it with the partitioning mode 
you want for the GPUs of that node. Currently, we support [MIG](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/) 
and [MPS](https://docs.nvidia.com/deploy/mps/index.html) partitioning for NVIDIA GPUs.

For instance, you can enable MIG partitioning on a node by labelling it as follows:
```bash
kubectl label nodes <node-name> "nos.nebuly.ai/gpu-partitioning=mig"
```

Please refer to the [Automatic GPU partitioning](doc/automatic-gpu-partitioning.md) documentation for the 
pre-requisites and the limitations of each supported partitioning mode.

### Create Elastic Resource Quotas
```yaml
$ kubectl apply -f - <<EOF 
apiVersion: nos.nebuly.ai/v1alpha1
kind: ElasticQuota
metadata:
  name: quota-a
  namespace: team-a
spec:
  min:
    cpu: 2
    nos.nebuly.ai/gpu-memory: 16
  max:
    cpu: 10
EOF
```

## Documentation

- [Automatic GPU partitioning](doc/automatic-gpu-partitioning.md)
  - [Getting started](doc/automatic-gpu-partitioning.md#getting-started)
  - [Enable nodes for automatic partitioning](doc/automatic-gpu-partitioning.md#enable-nodes-for-automatic-partitioning)
  - [MIG Partitioning](doc/automatic-gpu-partitioning.md#mig-partitioning)
  - [Configuration](doc/automatic-gpu-partitioning.md#configuration)
  - [Integration with nos scheduler](doc/automatic-gpu-partitioning.md#integration-with-nos-scheduler)
  - [Troubleshooting](doc/automatic-gpu-partitioning.md#troubleshooting)
- [Elastic Resource Quota management](doc/elastic-quota.md)
  - [Getting started](doc/elastic-quota.md#getting-started)
  - [How to define Resource Quotas](doc/elastic-quota.md#how-to-define-resource-quotas)
  - [Installation options](doc/elastic-quota.md#scheduler-installation-options)
  - [Troubleshooting](doc/elastic-quota.md#troubleshooting)

## Developer

- [Getting started](doc/developer/get-started.md)
- [Contribution guidelines](doc/developer/contribution-guidelines.md)
- [Roadmap]()

---

<p align="center">
  <a href="https://discord.gg/RbeQMu886J">Join the community</a>  | <a href="https://nebuly.gitbook.io/nebuly/welcome/questions-and-contributions"> Contribute </a>
</p>
