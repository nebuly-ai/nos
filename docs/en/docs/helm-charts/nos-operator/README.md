# nos-operator

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.1.0](https://img.shields.io/badge/AppVersion-0.1.0-informational?style=flat-square)

Install and manage `nos` custom resources.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.com> | <github.com/nebuly-ai> |
| Diego Fiori | <d.fiori@nebuly.com> | <github.com/diegofiori> |

## Source Code

* <https://github.com/nebuly-ai/nos>
* <https://github.com/nebuly-ai/helm-charts/nos-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Sets the affinity config of the operator Pod. |
| fullnameOverride | string | `""` |  |
| global.nvidiaGpuResourceMemoryGB | int | `32` | Defines how many GB of memory each nvidia.com/gpu resource has. |
| image.pullPolicy | string | `"IfNotPresent"` | Sets the operator Docker image pull policy. |
| image.repository | string | `"ghcr.io/nebuly-ai/nos-operator"` | Sets the operator Docker repository |
| image.tag | string | `""` | Overrides the operator Docker image tag whose default is the chart appVersion. |
| kubeRbacProxy | object | - | Configuration of the [Kube RBAC Proxy](https://github.com/brancz/kube-rbac-proxy), which runs as sidecar of the operator Pods. |
| leaderElection.enabled | bool | `true` | Enables/Disables the leader election of the operator controller manager. |
| logLevel | int | `0` | The level of log of the controller manager. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` | Sets the nodeSelector config of the operator Pod. |
| podAnnotations | object | `{}` | Sets the annotations of the operator Pod. |
| podSecurityContext | object | `{"runAsNonRoot":true}` | Sets the security context of the operator Pod. |
| replicaCount | int | `1` | Number of replicas of the controller manager Pod. |
| resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Sets the resource limits and requests of the operator controller manager container. |
| securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}` | Sets the security context of the operator container. |
| tolerations | list | `[]` | Sets the tolerations of the operator Pod. |

