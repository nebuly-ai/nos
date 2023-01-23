# nos

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.1.0](https://img.shields.io/badge/AppVersion-0.1.0-informational?style=flat-square)

The open-source platform for running AI workloads on k8s in an optimized way, both in terms of hardware utilization and workload performance.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.com> | <github.com/nebuly-ai> |
| Diego Fiori | <d.fiori@nebuly.com> | <github.com/diegofiori> |

## Source Code

* <https://github.com/nebuly-ai/nos>
* <https://github.com/nebuly-ai/nos/helm-charts/nos>
* <https://github.com/nebuly-ai/nos/helm-charts/nos-gpu-partitioner>

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| oci://ghcr.io/nebuly-ai/helm-charts | nos-gpu-partitioner | 0.1.0 |
| oci://ghcr.io/nebuly-ai/helm-charts | nos-operator | 0.1.0 |
| oci://ghcr.io/nebuly-ai/helm-charts | nos-scheduler | 0.1.0 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| allowDefaultNamespace | bool | `false` | If true allows to deploy `nos` chart in the `default` namespace |
| global.nvidiaGpuResourceMemoryGB | int | `32` | Defines how many GB of memory each nvidia.com/gpu resource has. |
| nos-gpu-partitioner | object | - | All values available [here](../nos-gpu-partitioner/README.md). |
| nos-gpu-partitioner.enabled | bool | `true` | Enable or disable the `nos gpu partitioner` |
| nos-operator | object | - | All values available [here](../nos-operator/README.md). |
| nos-operator.enabled | bool | `true` | Enable or disable the `nos operator` |
| nos-scheduler | object | - | All values available [here](../nos-scheduler/README.md). |
| nos-scheduler.enabled | bool | `true` | Enable or disable the `nos scheduler` |

