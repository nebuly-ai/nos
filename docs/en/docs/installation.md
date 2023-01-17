# Installation

You can install `nos` using Helm 3 (recommended).
You can find all the available configuration values in the Chart [documentation](helm-charts/nos/README.md).

```bash
helm install oci://ghcr.io/nebuly-ai/helm-charts/nos \
  --version 0.1.0 \
  --namespace nebuly-nos \
  --generate-name \
  --create-namespace
```

Alternatively, you can use Kustomize by cloning the repository and running `make deploy`.
