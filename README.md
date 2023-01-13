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

## Documentation

- [Automatic GPU partitioning](doc/automatic-gpu-partitioning.md)
  - [Overview](doc/automatic-gpu-partitioning.md#overview)
  - [Partitioning modes comparison](doc/automatic-gpu-partitioning.md#partitioning-modes-comparison)
  - [MIG Partitioning](doc/automatic-gpu-partitioning.md#mig-partitioning)
  - [MPS Partitioning](doc/automatic-gpu-partitioning.md#mps-partitioning)
  - [Configuration](doc/automatic-gpu-partitioning.md#configuration)
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
