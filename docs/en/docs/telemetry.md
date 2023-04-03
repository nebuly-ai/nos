# Sharing feedback to improve `nos`

Open source is a unique resource for sharing knowledge and building great projects collaboratively with the OSS
community. To support the development of `nos`, during the installation of you could share the information
strictly necessary to improve the features of this open-source project and facilitate bug detection and fixing.

More specifically, you will foster project enhancement by sharing details
about the setup and configuration of the environment where you are installing `nos` and its components.

**Which data do we collect?**

We make sure to collect as little data as possible to improve the open-source project:

- basic information about the Kubernetes cluster
    - Kubernetes version
    - Number of nodes
- basic information about each node of the cluster
    - Kubelet version
    - Operating system
    - Container runtime
    - Node resources
    - Labels from the [NVIDIA GPU Feature Discovery](https://github.com/NVIDIA/gpu-feature-discovery), if present
    - Label `node.kubernetes.io/instance-type`, if present
- configuration of `nos` components
    - values provided during the Helm chart installation

Please find below an example of telemetry collection:

```json
{
  "installationUUID": "feb0a960-ed22-4882-96cf-ef0b83deaeb1",
  "nodes": [
    {
      "name": "node-1",
      "Capacity": {
        "cpu": "5",
        "memory": "7111996Ki"
      },
      "Labels": {
        "nvidia.com/gpu": "true"
      },
      "NodeInfo": {
        "kernelVersion": "5.15.49-linuxkit",
        "osImage": "Ubuntu 22.04.1 LTS",
        "containerRuntimeVersion": "containerd://1.6.7",
        "kubeletVersion": "v1.24.4",
        "architecture": "arm64"
      }
    },
    {
      "name": "node-2",
      "Capacity": {
        "cpu": "2",
        "memory": "7111996Ki"
      },
      "Labels": null,
      "NodeInfo": {
        "kernelVersion": "5.15.49-linuxkit",
        "osImage": "Ubuntu 22.04.1 LTS",
        "containerRuntimeVersion": "containerd://1.6.7",
        "kubeletVersion": "v1.24.4",
        "architecture": "arm64"
      }
      "chartValues": {
        "allowDefaultNamespace": false,
        "global": {
          "nvidiaGpuResourceMemoryGB": 32
        }
      },
      "components": {
        "nos-gpu-partitioner": true,
        "nos-scheduler": true,
        "nos-operator": true
      }
    }
  ]
}
```

## How to opt-out?
You have two possibilities for opting-out:

1. Set the value `shareTelemetry` to false when installing `nos` with the Helm Chart
   ```bash
    helm install oci://ghcr.io/nebuly-ai/helm-charts/nos \
    --version 0.1.1 \
    --namespace nebuly-nos \
    --generate-name \
    --create-namespace \
    --set shareTelemetry=false
   ```
2. Install `nos` without using Helm


## Should I opt out?

Being open-source, we have very limited visibility into the use of the tool unless someone actively contacts us or opens
an issue on GitHub.

We would appreciate it if you would maintain telemetry, as it helps us improve the source code. In fact, it brings
increasing value to the project and helps us to better prioritize feature development.

We understand that you may still prefer not to share telemetry data and we respect that desire. Please follow the steps
above to disable data collection.