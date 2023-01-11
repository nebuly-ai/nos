# Nebuly Operating System (nos)

`nos` is the open-source module for running AI workloads on Kubernetes in an optimized way, both in terms of
hardware utilization and workload performance.

The operating system layer is responsible for workloads scheduling and hardware abstraction.
It orchestrates the workloads taking into account considerations specific for AI/ML workloads and leveraging
techniques typical of High-performance Computing (HPC), and it hides the underlying hardware complexities.

Currently, this layer provides two features [Automatic GPU partitioning](doc/automatic-gpu-partitioning.md) and
[Elastic Resource Quota management](doc/elastic-quota.md).

## Documentation

- [Automatic GPU partitioning](doc/automatic-gpu-partitioning.md)
  - [Getting started](doc/automatic-gpu-partitioning.md#getting-started)
  - [Enable nodes for automatic partitioning](doc/automatic-gpu-partitioning.md#enable-nodes-for-automatic-partitioning)
  - [MIG Partitioning](doc/automatic-gpu-partitioning.md#mig-partitioning)
  - [Configuration](doc/automatic-gpu-partitioning.md#configuration)
  - [Integration with nos scheduler](doc/automatic-gpu-partitioning.md#integration-with-nebulnetes-scheduler)
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
