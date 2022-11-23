# Nebulnetes (n8s)

Nebulnetes is the open-source platform for running your AI workloads on Kubernetes maximizing hardware utilization.

## Product design

Nebulnetes is designed around 4 layers, each focusing on specific aspects of the stack required 
for running AI workloads:

1. AI Application
2. Gateway
3. Operating system
4. Hardware

## Operating system 

The operating system layer is responsible for workloads scheduling and hardware abstraction. 
Currently, this layer provides two main features:
- [Automatic GPU partitioning](doc/automatic-gpu-partitioning.md)
- [Elastic Resource Quota management](doc/elastic-quota.md)

## Documentation

- [Automatic GPU partitioning](doc/automatic-gpu-partitioning.md)
  - [Getting started](doc/automatic-gpu-partitioning.md#getting-started)
  - [Configuration](doc/automatic-gpu-partitioning.md#configuration)
  - [Integration with Elastic Resource quota](doc/automatic-gpu-partitioning.md#integration-with-nebulnetes-scheduler)
- [Elastic Resource Quota management](doc/elastic-quota.md)
  - [Getting started](doc/elastic-quota.md#getting-started)
  - [How to define Resource Quotas](doc/elastic-quota.md#how-to-define-resource-quotas)
  - [Installation options](doc/elastic-quota.md#scheduler-installation-options)

---

<p align="center">
  <a href="https://discord.gg/RbeQMu886J">Join the community</a> 
</p>
