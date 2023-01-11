# nos

![Version: 0.0.1-alpha.3](https://img.shields.io/badge/Version-0.0.1--alpha.3-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.1-alpha.3](https://img.shields.io/badge/AppVersion-0.0.1--alpha.3-informational?style=flat-square)

The open-source platform for running AI workloads on k8s in an optimized way, both in terms of hardware utilization and workload performance.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.ai> | <github.com/Telemaco019> |
| Diego Fiori | <d.fiori@nebuly.ai> | <github.com/diegofiori> |

## Source Code

* <https://github.com/nebuly-ai/nos>
* <https://github.com/nebuly-ai/nos/helm-charts/nos>
* <https://github.com/nebuly-ai/nos/helm-charts/nos-gpu-partitioner>

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| oci://ghcr.io/nebuly-ai/helm-charts | nos-gpu-partitioner | 0.0.1-alpha.3 |
| oci://ghcr.io/nebuly-ai/helm-charts | nos-operator | 0.0.1-alpha.3 |
| oci://ghcr.io/nebuly-ai/helm-charts | nos-scheduler | 0.0.1-alpha.3 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| global.nvidiaGpuResourceMemoryGB | int | `32` | Defines how many GB of memory each nvidia.com/gpu resource has. |
| gpu-partitioner | object | - | Config of the GPU Partitioner component. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/gpu-partitioner). |
| gpu-partitioner.enabled | bool | `true` | Enable or disable the GPU Partitioner |
| operator | object | - | Config of the Nebulnetes operator. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/n8s-operator). |
| operator.enabled | bool | `true` | Enable or disable the Nebulnetes Operator |
| scheduler | object | - | Config of the Nebulnetes scheduler. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/scheduler). |
| scheduler.enabled | bool | `true` | Enable or disable the Nebulnetes Scheduler |

