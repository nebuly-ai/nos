# gpu-partitioner

![Version: 0.0.1-alpha.3](https://img.shields.io/badge/Version-0.0.1--alpha.3-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.1-alpha.3](https://img.shields.io/badge/AppVersion-0.0.1--alpha.3-informational?style=flat-square)

Automatically partitions GPUs exposing them to Kubernetes as multiple resources (slices).

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.ai> | <github.com/Telemaco019> |
| Diego Fiori | <d.fiori@nebuly.ai> | <github.com/diegofiori> |

## Source Code

* <https://github.com/Telemaco019/nebulnetes>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Sets the affinity config of the GPU Partitioner Pod. |
| batchWindowIdleSeconds | int | `10` | Idle seconds before the GPU partitioner processes the current batch if no new pending Pods are created, and the timeout has not been reached.  Higher values make the GPU partitioner will potentially take into account more pending Pods when deciding the GPU partitioning plan, but the partitioning will be performed less frequently |
| batchWindowTimeoutSeconds | int | `60` | Timeout of the window used by the GPU partitioner for batching pending Pods.  Higher values make the GPU partitioner will potentially take into account more pending Pods when deciding the GPU partitioning plan, but the partitioning will be performed less frequently |
| devicePlugin.config.name | string | `"nvidia-plugin-configs"` | Name of the ConfigMap containing the NVIDIA Device Plugin configuration files. It must be equal to the value "devicePlugin.config.name" of the Helm chart used for deploying the NVIDIA GPU Operator. |
| devicePlugin.config.namespace | string | `"gpu-operator"` | Namespace of the ConfigMap containing the NVIDIA Device Plugin configuration files. It must be equal to the namespace where the NVIDIA Device Plugin has been deployed to. |
| image.pullPolicy | string | `"IfNotPresent"` | Sets the GPU Partitioner Docker image pull policy. |
| image.repository | string | `"ghcr.io/telemaco019/nebulnetes-gpu-partitioner"` | Sets the GPU Partitioner Docker image. |
| image.tag | string | `""` | Overrides the GPU Partitioner image tag whose default is the chart appVersion. |
| knownMigGeometries | object | - | Map that associates to each GPU model its possible MIG configurations |
| kubeRbacProxy | object | - | Configuration of the [Kube RBAC Proxy](https://github.com/brancz/kube-rbac-proxy), which runs as sidecar of all the GPU Partitioner components Pods. |
| leaderElection.enabled | bool | `true` | Enables/Disables the leader election of the GPU Partitioner controller manager. |
| logLevel | int | `0` | The level of log of the GPU Partitioner. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| migAgent | object | - | Configuration of the MIG Agent component of the GPU Partitioner. |
| migAgent.image.pullPolicy | string | `"IfNotPresent"` | Sets the MIG Agent Docker image pull policy. |
| migAgent.image.repository | string | `"ghcr.io/telemaco019/nebulnetes-mig-agent"` | Sets the MIG Agent Docker image. |
| migAgent.image.tag | string | `""` | Overrides the MIG Agent image tag whose default is the chart appVersion. |
| migAgent.logLevel | int | `0` | The level of log of the MIG Agent. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| migAgent.reportConfigIntervalSeconds | int | `10` | Interval at which the mig-agent will report to k8s the MIG partitioning status of the GPUs of the Node |
| migAgent.resources | object | `{"limits":{"cpu":"100m","memory":"128Mi"}}` | Sets the resource requests and limits of the MIG Agent container. |
| migAgent.tolerations | list | `[{"effect":"NoSchedule","key":"kubernetes.azure.com/scalesetpriority","operator":"Equal","value":"spot"}]` | Sets the tolerations of the MIG Agent Pod. |
| nodeSelector | object | `{}` | Sets the nodeSelector config of the GPU Partitioner Pod. |
| podAnnotations | object | `{}` | Sets the annotations of the GPU Partitioner Pod. |
| podSecurityContext | object | `{"runAsNonRoot":true,"runAsUser":1000}` | Sets the security context of the GPU partitioner Pod. |
| replicaCount | int | `1` | Number of replicas of the gpu-manager Pod. |
| resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Sets the resource limits and requests of the GPU partitioner container. |
| scheduler.config.name | string | `"n8s-scheduler-config"` | Name of the ConfigMap containing the k8s scheduler configuration file. If not specified or the ConfigMap does not exist, the GPU partitioner will use the default k8s scheduler profile. |
| timeSlicingAgent | object | - | Configuration of the Time Slicing Agent component of the GPU Partitioner. |
| timeSlicingAgent.image.pullPolicy | string | `"IfNotPresent"` | Sets the Time Slicing Agent Docker image pull policy. |
| timeSlicingAgent.image.repository | string | `"ghcr.io/telemaco019/nebulnetes-time-slicing-agent"` | Sets the Time Slicing Agent Docker image. |
| timeSlicingAgent.image.tag | string | `"latest"` | Overrides the Time Slicing Agent image tag whose default is the chart appVersion. |
| timeSlicingAgent.logLevel | int | `0` | The level of log of the Time Slicing Agent. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| timeSlicingAgent.reportConfigIntervalSeconds | int | `10` | Interval at which the mig-agent will report to k8s the MIG partitioning status of the GPUs of the Node |
| timeSlicingAgent.resources | object | `{"limits":{"cpu":"100m","memory":"128Mi"}}` | Sets the resource requests and limits of the Time Slicing Agent container. |
| timeSlicingAgent.tolerations | list | `[{"effect":"NoSchedule","key":"kubernetes.azure.com/scalesetpriority","operator":"Equal","value":"spot"}]` | Sets the tolerations of the Time Slicing Agent Pod. |
| tolerations | list | `[]` | Sets the tolerations of the GPU Partitioner Pod. |

