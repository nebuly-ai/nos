# nebulnetes

![Version: 0.0.1-alpha.3](https://img.shields.io/badge/Version-0.0.1--alpha.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.1-alpha.3](https://img.shields.io/badge/AppVersion-0.0.1--alpha.2-informational?style=flat-square)

The open-source platform for running AI workloads on k8s in an optimized way, both in terms of hardware utilization and workload performance.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.ai> | <github.com/Telemaco019> |
| Diego Fiori | <d.fiori@nebuly.ai> | <github.com/diegofiori> |

## Source Code

* <https://github.com/Telemaco019/nebulnetes>
* <https://github.com/Telemaco019/nebulnetes/helm-charts/nebulnetes>
* <https://github.com/Telemaco019/nebulnetes/helm-charts/gpu-partitioner>

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| oci://ghcr.io/telemaco019/helm-charts | gpu-partitioner | 0.0.1-alpha.3 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| gpu-partitioner | object | - | Config of the GPU Partitioner component. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/gpu-partitioner). |
| gpu-partitioner.enabled | bool | `true` | Enable or disable the GPU Partitioner component |
| namePrefix | string | `"n8s"` | The prefix used for generating all the resource names. |
| nvidiaGpuResourceMemoryGB | int | `32` | Defines how much GB of memory does a nvidia.com/gpu has. |
| operator | object | - | Config of the Nebulnetes operator. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/n8s-operator). |
| operator.affinity | object | `{}` | Sets the affinity config of the operator Pod. |
| operator.enabled | bool | `true` | Enable or disable the Nebulnetes Operator |
| operator.image.pullPolicy | string | `"IfNotPresent"` | Sets the operator Docker image pull policy. |
| operator.image.repository | string | `"ghcr.io/telemaco019/nebulnetes-operator"` | Sets the operator Docker repository |
| operator.image.tag | string | `""` | Overrides the operator Docker image tag whose default is the chart appVersion. |
| operator.kubeRbacProxy | object | - | Configuration of the [Kube RBAC Proxy](https://github.com/brancz/kube-rbac-proxy), which runs as sidecar of the operator Pods. |
| operator.leaderElection.enabled | bool | `true` | Enables/Disables the leader election of the operator controller manager. |
| operator.logLevel | int | `0` | The level of log of the controller manager. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| operator.nodeSelector | object | `{}` | Sets the nodeSelector config of the operator Pod. |
| operator.podAnnotations | object | `{}` | Sets the annotations of the operator Pod. |
| operator.podSecurityContext | object | `{"runAsNonRoot":true,"runAsUser":1000}` | Sets the security context of the operator Pod. |
| operator.replicaCount | int | `1` | Number of replicas of the controller manager Pod. |
| operator.resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Sets the resource limits and requests of the operator controller manager container. |
| operator.tolerations | list | `[]` | Sets the tolerations of the operator Pod. |
| scheduler | object | - | Config of the Nebulnetes scheduler. |
| scheduler.affinity | object | `{}` | Sets the affinity config of the scheduler deployment. |
| scheduler.config | object | `{}` | Overrides the Kube Scheduler configuration |
| scheduler.enabled | bool | `true` | Enable or disable the Nebulnetes Scheduler |
| scheduler.image.pullPolicy | string | `"IfNotPresent"` | Sets the scheduler Docker image pull policy. |
| scheduler.image.repository | string | `"ghcr.io/telemaco019/nebulnetes-scheduler"` | Sets the scheduler Docker image. |
| scheduler.image.tag | string | `""` | Overrides the scheduler image tag whose default is the chart appVersion. |
| scheduler.leaderElection.enabled | bool | `true` | Enables/Disables the leader election of the scheduler when deployed with multiple replicas. |
| scheduler.logLevel | int | `0` | The level of log of the scheduler. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| scheduler.nodeSelector | object | `{}` | Sets the nodeSelector config of the scheduler deployment. |
| scheduler.podAnnotations | object | `{}` | Sets the annotations of the scheduler Pod. |
| scheduler.replicaCount | int | `1` | Number of replicas of the controller manager Pod. |
| scheduler.resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Sets the resource limits and requests of the operator controller manager container. |
| scheduler.tolerations | list | `[]` | Sets the tolerations of the scheduler deployment. |

