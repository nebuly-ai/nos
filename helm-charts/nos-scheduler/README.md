# nos-scheduler

![Version: 0.0.1-alpha.3](https://img.shields.io/badge/Version-0.0.1--alpha.3-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.1-alpha.3](https://img.shields.io/badge/AppVersion-0.0.1--alpha.3-informational?style=flat-square)

Kubernetes scheduler optimized for managing AI workloads.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.ai> | <github.com/nebuly-ai> |
| Diego Fiori | <d.fiori@nebuly.ai> | <github.com/diegofiori> |

## Source Code

* <https://github.com/nebuly-ai/nos>
* <https://github.com/nebuly-ai/helm-charts/nos-scheduler>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Sets the affinity config of the scheduler deployment. |
| config | object | `{}` | Overrides the Kube Scheduler configuration |
| fullnameOverride | string | `""` |  |
| global.nvidiaGpuResourceMemoryGB | int | `32` | Defines how many GB of memory each nvidia.com/gpu resource has. |
| image.pullPolicy | string | `"IfNotPresent"` | Sets Docker image pull policy. |
| image.repository | string | `"ghcr.io/nebuly-ai/nos-scheduler"` | Sets Docker image. |
| image.tag | string | `""` | Overrides the image tag whose default is the chart appVersion. |
| leaderElection.enabled | bool | `true` | Enables/Disables the leader election when deployed with multiple replicas. |
| logLevel | int | `0` | The level of log of the scheduler. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` | Sets the nodeSelector config of the scheduler deployment. |
| podAnnotations | object | `{}` | Sets the annotations of the scheduler Pod. |
| podSecurityContext | object | `{}` | Sets the security context of the scheduler Pod |
| replicaCount | int | `1` | Number of replicas of the scheduler. |
| resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Sets the resource limits and requests of the scheduler container. |
| securityContext | object | `{"privileged":false}` | Sets the security context of the scheduler container |
| tolerations | list | `[]` | Sets the tolerations of the scheduler deployment. |

