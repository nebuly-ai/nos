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

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| gpu-partitioner | object | `{"enabled":true}` | Config of the GPU Partitioner component. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/gpu-partitioner). |
| gpu-partitioner.enabled | bool | `true` | Enable or disable the GPU Partitioner component |
| n8s-operator | object | `{"affinity":{},"enabled":true,"image":{"pullPolicy":"IfNotPresent","repository":"ghcr.io/telemaco019/nebulnetes-operator","tag":""},"kubeRbacProxy":{"image":{"pullPolicy":"IfNotPresent","repository":"gcr.io/kubebuilder/kube-rbac-proxy","tag":"v0.13.0"},"logLevel":1,"resources":{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"5m","memory":"64Mi"}}},"leaderElection":{"enabled":true},"logLevel":0,"nodeSelector":{},"nvidiaGpuResourceMemoryGB":32,"podAnnotations":{},"podSecurityContext":{"runAsNonRoot":true,"runAsUser":1000},"replicaCount":1,"resources":{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}},"tolerations":[]}` | Config of the Nebulnetes operator. All possible values available [here](https://github.com/Telemaco019/nebulnetes/tree/main/helm-charts/n8s-operator). |
| n8s-operator.affinity | object | `{}` | Sets the affinity config of the operator Pod. |
| n8s-operator.enabled | bool | `true` | Enable or disable the Nebulnetes Operator |
| n8s-operator.image.pullPolicy | string | `"IfNotPresent"` | Sets the operator Docker image pull policy. |
| n8s-operator.image.repository | string | `"ghcr.io/telemaco019/nebulnetes-operator"` | Sets the operator Docker repository |
| n8s-operator.image.tag | string | `""` | Overrides the operator Docker image tag whose default is the chart appVersion. |
| n8s-operator.kubeRbacProxy | object | - | Configuration of the [Kube RBAC Proxy](https://github.com/brancz/kube-rbac-proxy), which runs as sidecar of the operator Pods. |
| n8s-operator.leaderElection.enabled | bool | `true` | Enables/Disables the leader election of the operator controller manager. |
| n8s-operator.logLevel | int | `0` | The level of log of the controller manager. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| n8s-operator.nodeSelector | object | `{}` | Sets the nodeSelector config of the operator Pod. |
| n8s-operator.nvidiaGpuResourceMemoryGB | int | `32` | Defines how much GB of memory does a nvidia.com/gpu has. |
| n8s-operator.podAnnotations | object | `{}` | Sets the annotations of the operator Pod. |
| n8s-operator.podSecurityContext | object | `{"runAsNonRoot":true,"runAsUser":1000}` | Sets the security context of the operator Pod. |
| n8s-operator.replicaCount | int | `1` | Number of replicas of the controller manager Pod. |
| n8s-operator.resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Sets the resource limits and requests of the operator controller manager container. |
| n8s-operator.tolerations | list | `[]` | Sets the tolerations of the operator Pod. |
| n8s-scheduler | object | `{"affinity":{},"config":{},"enabled":true,"image":{"pullPolicy":"IfNotPresent","repository":"ghcr.io/telemaco019/nebulnetes-scheduler","tag":""},"leaderElection":{"enabled":true},"logLevel":0,"nodeSelector":{},"nvidiaGpuResourceMemoryGB":32,"podAnnotations":{},"replicaCount":1,"resources":{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}},"tolerations":[]}` | Config of the Nebulnetes scheduler. |
| n8s-scheduler.affinity | object | `{}` | Sets the affinity config of the scheduler deployment. |
| n8s-scheduler.config | object | `{}` | Overrides the Kube Scheduler configuration |
| n8s-scheduler.enabled | bool | `true` | Enable or disable the Nebulnetes Scheduler |
| n8s-scheduler.image.pullPolicy | string | `"IfNotPresent"` | Sets the scheduler Docker image pull policy. |
| n8s-scheduler.image.repository | string | `"ghcr.io/telemaco019/nebulnetes-scheduler"` | Sets the scheduler Docker image. |
| n8s-scheduler.image.tag | string | `""` | Overrides the scheduler image tag whose default is the chart appVersion. |
| n8s-scheduler.leaderElection.enabled | bool | `true` | Enables/Disables the leader election of the scheduler when deployed with multiple replicas. |
| n8s-scheduler.logLevel | int | `0` | The level of log of the scheduler. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| n8s-scheduler.nodeSelector | object | `{}` | Sets the nodeSelector config of the scheduler deployment. |
| n8s-scheduler.nvidiaGpuResourceMemoryGB | int | `32` | Defines how much GB of memory does a nvidia.com/gpu has. |
| n8s-scheduler.podAnnotations | object | `{}` | Sets the annotations of the scheduler Pod. |
| n8s-scheduler.replicaCount | int | `1` | Number of replicas of the controller manager Pod. |
| n8s-scheduler.resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Sets the resource limits and requests of the operator controller manager container. |
| n8s-scheduler.tolerations | list | `[]` | Sets the tolerations of the scheduler deployment. |

