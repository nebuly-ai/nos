# nebulnetes

![Version: 0.0.1-alpha.2](https://img.shields.io/badge/Version-0.0.1--alpha.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.1-alpha.2](https://img.shields.io/badge/AppVersion-0.0.1--alpha.2-informational?style=flat-square)

The open-source platform for running AI workloads on k8s in an optimized way, both in terms of hardware utilization and workload performance.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.ai> | <github.com/Telemaco019> |
| Diego Fiori | <d.fiori@nebuly.ai> | <github.com/diegofiori> |

## Source Code

* <https://github.com/Telemaco019/nebulnetes>
* <https://github.com/Telemaco019/nebulnetes/helm-charts/gpu-partitioner>
* <https://github.com/Telemaco019/nebulnetes/helm-charts/n8s-scheduler>
* <https://github.com/Telemaco019/nebulnetes/helm-charts/n8s-operator>

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| oci://ghcr.io/telemaco019/helm-charts | gpu-partitioner | 0.0.1-alpha.2 |
| oci://ghcr.io/telemaco019/helm-charts | n8s-operator | 0.0.1-alpha.2 |
| oci://ghcr.io/telemaco019/helm-charts | n8s-scheduler | 0.0.1-alpha.2 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| gpu-partitioner | object | `{"enabled":true}` | Config of the GPU Partitioner component. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/gpu-partitioner). |
| gpu-partitioner.enabled | bool | `true` | Enable or disable the GPU Partitioner component |
| n8s-operator | object | `{"enabled":true}` | Config of the Nebulnetes operator. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/n8s-operator). |
| n8s-operator.enabled | bool | `true` | Enable or disable the Nebulnetes Operator |
| n8s-scheduler | object | `{"enabled":true}` | Config of the Nebulnetes scheduler. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/n8s-scheduler). |
| n8s-scheduler.enabled | bool | `true` | Enable or disable the Nebulnetes Scheduler |

