# gpu-partitioner

![Version: 0.0.1-alpha.2](https://img.shields.io/badge/Version-0.0.1--alpha.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.1-alpha.2](https://img.shields.io/badge/AppVersion-0.0.1--alpha.2-informational?style=flat-square)

Automatically partitions GPUs exposing them to Kubernetes as multiple resources (slices).

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.ai> | <github.com/Telemaco019> |
| Diego Fiori | <d.fiori@nebuly.ai> | <github.com/diegofiori> |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| batchWindowIdleSeconds | int | `10` | Idle seconds before the GPU partitioner processes the current batch if no new pending Pods are created, and the timeout has not been reached.  Higher values make the GPU partitioner will potentially take into account more pending Pods when deciding the GPU partitioning plan, but the partitioning will be performed less frequently |
| batchWindowTimeoutSeconds | int | `60` | Timeout of the window used by the GPU partitioner for batching pending Pods.  Higher values make the GPU partitioner will potentially take into account more pending Pods when deciding the GPU partitioning plan, but the partitioning will be performed less frequently |
| devicePlugin | object | `{"config":{"name":"nvidia-plugin-configs","namespace":"gpu-operator"}}` | Namespaced name of the ConfigMap containing the NVIDIA Device Plugin configuration files. It must be equal to the value "devicePlugin.config.name" of the Helm chart used for deploying the NVIDIA GPU Operator. |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/telemaco019/nebulnetes-gpu-partitioner"` |  |
| image.tag | string | `"latest"` |  |
| knownMigGeometries | string | The default map does not include all the possible NVIDIA GPU model, so it might not include the exact | Map that associates to each GPU model its possible MIG configurations models of the GPUs of your node (exposed by the label `nvidia.com/gpu.product`). |
| kubeRbacProxy.image.pullPolicy | string | `"IfNotPresent"` |  |
| kubeRbacProxy.image.repository | string | `"gcr.io/kubebuilder/kube-rbac-proxy"` |  |
| kubeRbacProxy.image.tag | string | `"v0.13.0"` |  |
| kubeRbacProxy.logLevel | int | `1` |  |
| kubeRbacProxy.resources.limits.cpu | string | `"500m"` |  |
| kubeRbacProxy.resources.limits.memory | string | `"128Mi"` |  |
| kubeRbacProxy.resources.requests.cpu | string | `"5m"` |  |
| kubeRbacProxy.resources.requests.memory | string | `"64Mi"` |  |
| leaderElection | object | `{"enabled":true}` | Controller manager leader election |
| logLevel | int | `1` |  |
| migAgent.image.pullPolicy | string | `"IfNotPresent"` |  |
| migAgent.image.repository | string | `"ghcr.io/telemaco019/nebulnetes-mig-agent"` |  |
| migAgent.image.tag | string | `""` |  |
| migAgent.logLevel | int | `1` |  |
| migAgent.reportConfigIntervalSeconds | int | `10` | Interval at which the mig-agent will report to k8s the MIG partitioning status of the GPUs of the Node |
| migAgent.resources.limits.cpu | string | `"100m"` |  |
| migAgent.resources.limits.memory | string | `"128Mi"` |  |
| migAgent.tolerations[0].effect | string | `"NoSchedule"` |  |
| migAgent.tolerations[0].key | string | `"kubernetes.azure.com/scalesetpriority"` |  |
| migAgent.tolerations[0].operator | string | `"Equal"` |  |
| migAgent.tolerations[0].value | string | `"spot"` |  |
| namePrefix | string | `"n8s"` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| podSecurityContext.runAsUser | int | `1000` |  |
| replicaCount | int | `1` |  |
| resources.limits.cpu | string | `"500m"` |  |
| resources.limits.memory | string | `"128Mi"` |  |
| resources.requests.cpu | string | `"10m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| scheduler.config | object | `{"name":"n8s-scheduler-config"}` | Name of the ConfigMap containing the k8s scheduler configuration file. If not specified or the ConfigMap does not exist, the GPU partitioner will use the default k8s scheduler profile. |
| timeSlicingAgent.image.pullPolicy | string | `"IfNotPresent"` |  |
| timeSlicingAgent.image.repository | string | `"ghcr.io/telemaco019/nebulnetes-time-slicing-agent"` |  |
| timeSlicingAgent.image.tag | string | `"latest"` |  |
| timeSlicingAgent.logLevel | int | `1` |  |
| timeSlicingAgent.reportConfigIntervalSeconds | int | `10` | Interval at which the mig-agent will report to k8s the MIG partitioning status of the GPUs of the Node |
| timeSlicingAgent.resources.limits.cpu | string | `"100m"` |  |
| timeSlicingAgent.resources.limits.memory | string | `"128Mi"` |  |
| timeSlicingAgent.tolerations[0].effect | string | `"NoSchedule"` |  |
| timeSlicingAgent.tolerations[0].key | string | `"kubernetes.azure.com/scalesetpriority"` |  |
| timeSlicingAgent.tolerations[0].operator | string | `"Equal"` |  |
| timeSlicingAgent.tolerations[0].value | string | `"spot"` |  |
| tolerations | list | `[]` |  |

