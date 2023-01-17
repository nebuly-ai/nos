# Nebuly Operating System (nos)

`nos` is the open-source module for running AI workloads on Kubernetes in an optimized way, both in terms of
hardware utilization and workload performance.

The operating system layer is responsible for workloads scheduling and hardware abstraction.
It orchestrates the workloads taking into account considerations specific for AI/ML workloads and leveraging
techniques typical of High-performance Computing (HPC), and it hides the underlying hardware complexities.

Currently, this layer provides two features [Automatic GPU partitioning](docs/automatic-gpu-partitioning.md) and
[Elastic Resource Quota management](docs/elastic-quota.md).

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

* [Automatic GPU partitioning](docs/en/docs/automatic-gpu-partitioning.md)
  * [Overview](docs/en/docs/automatic-gpu-partitioning.md#overview)
  * [Partitioning modes comparison](docs/en/docs/automatic-gpu-partitioning.md#partitioning-modes-comparison)
  * [MIG Partitioning](docs/en/docs/automatic-gpu-partitioning.md#mig-partitioning)
  * [MPS Partitioning](docs/en/docs/automatic-gpu-partitioning.md#mps-partitioning)
  * [Configuration](docs/en/docs/automatic-gpu-partitioning.md#configuration)
  * [Troubleshooting](docs/en/docs/automatic-gpu-partitioning.md#troubleshooting)
* [Elastic Resource Quota management](docs/en/docs/elastic-quota.md)
  * [Getting started](docs/en/docs/elastic-quota.md#getting-started)
  * [How to define Resource Quotas](docs/en/docs/elastic-quota.md#how-to-define-resource-quotas)
  * [Installation options](docs/en/docs/elastic-quota.md#scheduler-installation-options)
  * [Troubleshooting](docs/en/docs/elastic-quota.md#troubleshooting)

## Developer

* [Getting started](docs/developer/get-started.md)
* [Contribution guidelines](docs/developer/contribution-guidelines.md)
* [Roadmap]()

---

<p align="center">
  <a href="https://discord.gg/RbeQMu886J">Join the community</a>  | <a href="https://nebuly.gitbook.io/nebuly/welcome/questions-and-contributions"> Contribute </a>
</p>
