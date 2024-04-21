# Nebuly Operating System (nos)

![](docs/en/docs/img/nos-logo.png)

---

**Documentation**: <a href="https://nebuly-ai.github.io/nos/overview" target="_blank"> docs.nebuly.com/nos/overview </a>

If you like the project please support it by leaving a star ✨

---

`nos` is the open-source module to efficiently run AI workloads on Kubernetes,
increasing GPU utilization, cutting down infrastructure costs and improving workloads performance.

Currently, the available features are:

* [Dynamic GPU partitioning](https://nebuly-ai.github.io/nos/dynamic-gpu-partitioning/overview): allow to schedule Pods requesting
fractions of GPU. GPU partitioning is performed automatically in real-time based on the Pods pending and running in
the cluster, so that Pods can request only the resources that are strictly necessary and GPUs are always fully utilized.

* [Elastic Resource Quota management](https://nebuly-ai.github.io/nos/elastic-resource-quota/overview): increase the number of Pods running on the
cluster by allowing namespaces to borrow quotas of reserved resources from other namespaces as long as they are
not using them.

![](docs/en/docs/img/gpu-utilization.png)


## Getting started

### Prerequisites

* Kubernetes v1.23 or newer
* [GPU Support must be enabled](http://nebuly-ai.github.io/nos/prerequisites/#enable-gpu-support)
* [Nebuly k8s-device-plugin](https://github.com/nebuly-ai/k8s-device-plugin) (optional, required only if you want to enable MPS partitioning)
* [cert-manager](https://cert-manager.io/docs/) (optional, but recommended)


### Installation

You can install `nos` using Helm 3 (recommended).
You can find all the available configuration values in the Chart [documentation](https://nebuly-ai.github.io/nos/helm-charts/nos/).

```bash
helm install oci://ghcr.io/nebuly-ai/helm-charts/nos \
  --version 0.1.2 \
  --namespace nebuly-nos \
  --generate-name \
  --create-namespace
```

Alternatively, you can use Kustomize by cloning the repository and running `make deploy`.
