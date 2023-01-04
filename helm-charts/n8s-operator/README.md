# n8s-operator

![Version: 0.0.1-alpha.2](https://img.shields.io/badge/Version-0.0.1--alpha.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.1-alpha.2](https://img.shields.io/badge/AppVersion-0.0.1--alpha.2-informational?style=flat-square)

Custom Resource Definitions (CRDs) and controllers of Nebulnetes.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Michele Zanotti | <m.zanotti@nebuly.ai> | <github.com/Telemaco019> |
| Diego Fiori | <d.fiori@nebuly.ai> | <github.com/diegofiori> |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Sets the affinity config of the operator Pod. |
| image.pullPolicy | string | `"IfNotPresent"` | Sets the operator Docker image pull policy. |
| image.repository | string | `"ghcr.io/telemaco019/nebulnetes-operator"` | Sets the operator Docker repository |
| image.tag | string | `""` | Overrides the operator Docker image tag whose default is the chart appVersion. |
| kubeRbacProxy | object | - | Configuration of the [Kube RBAC Proxy](https://github.com/brancz/kube-rbac-proxy), which runs as sidecar of the operator Pods. |
| leaderElection.enabled | bool | `true` | Enables/Disables the leader election of the operator controller manager. |
| logLevel | int | `0` | The level of log of the controller manager. Zero corresponds to `info`, while values greater or equal than 1 corresponds to higher debug levels. **Must be >= 0**. |
| namePrefix | string | `"n8s"` | The prefix used for generating all the resource names. |
| nodeSelector | object | `{}` | Sets the nodeSelector config of the operator Pod. |
| nvidiaGpuResourceMemoryGB | int | `32` | Defines how much GB of memory does a nvidia.com/gpu has. |
| podAnnotations | object | `{}` | Sets the annotations of the operator Pod. |
| podSecurityContext | object | `{"runAsNonRoot":true,"runAsUser":1000}` | Sets the security context of the operator Pod. |
| replicaCount | int | `1` | Number of replicas of the controller manager Pod. |
| resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Sets the resource limits and requests of the operator controller manager container. |
| tolerations | list | `[]` | Sets the tolerations of the operator Pod. |

