LOCALBIN ?= $(shell pwd)/bin
KIND_CLUSTER ?= demo-kubeflow
KUBEFLOW_PIPELINE_VERSION ?= 1.8.2
SYSTEM_ARCH ?= $(shell uname -sm | sed -e "s/ /-/g" |  tr '[:upper:]' '[:lower:]')


# Location to install dependencies to
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Kind cluster
$(LOCALBIN)/.setup: $(LOCALBIN)
	kind create cluster --name $(KIND_CLUSTER) --config kind.yaml
	touch $@





.PHONY: deploy
deploy: $(LOCALBIN)/.setup
	# Install KubeFlow Pipelines
	kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/cluster-scoped-resources?ref=$(KUBEFLOW_PIPELINE_VERSION)"
	kubectl wait --for condition=established --timeout=60s crd/applications.app.k8s.io
	kubectl apply -k "github.com/kubeflow/pipelines/manifests/kustomize/env/platform-agnostic-pns?ref=$(KUBEFLOW_PIPELINE_VERSION)"


.PHONY: port-forward
port-forward:
	kubectl port-forward -n kubeflow svc/ml-pipeline-ui 8080:80


clean:
	kind delete cluster --name $(KIND_CLUSTER)
	rm -rf $(LOCALBIN)
