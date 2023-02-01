# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

DOCKER_REGISTRY ?= ghcr.io/telemaco019

# Image URLs to build/push Docker image targets
CLIENT_IMG ?= $(DOCKER_REGISTRY)/demo-cifar10-client:latest


##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
ISTIOCTL ?= $(LOCALBIN)/istioctl
KUSTOMIZE ?= $(LOCALBIN)/kustomize


## Tool Versions
ISTIOCTL_VERSION ?= 1.16.2
KUSTOMIZE_VERSION ?= v4.5.5


.PHONY: istioctl
istioctl: $(ISTIOCTL) ## Download istioctl locally if necessary.
$(ISTIOCTL): $(LOCALBIN)
	test -s $(ISTIOCTL) || ( curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$(ISTIOCTL_VERSION) sh - && mv istio-$(ISTIOCTL_VERSION)/bin/istioctl $(ISTIOCTL) )

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }


##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: port-forward-prometheus
port-forward-prometheus:
	kubectl port-forward -n nos-monitoring svc/nos-monitoring-prometheus 9090:9090

##@ Components installation

## Components versions
NVIDIA_GPU_OPERATOR_VERSION ?= v22.9.0
NOS_VERSION ?= 0.1.0
SELDON_CORE_VERSION ?= v1.15.0

.PHONY: install-istio
install-istio: istioctl
	$(ISTIOCTL) install --set profile=demo -y

.PHONY: install-cert-manager
install-cert-manager:
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.10.0/cert-manager.yaml

.PHONY: install-nebuly-device-plugin
install-nebuly-device-plugin:
	helm upgrade --install nebuly-devcice-plugin \
	  oci://ghcr.io/nebuly-ai/helm-charts/nvidia-device-plugin \
	  --version 0.13.0 \
	  -n nebuly-nvidia \
	  --create-namespace \

.PHONY: install-nos
install-nos:
	helm upgrade --install nebuly-nos oci://ghcr.io/nebuly-ai/helm-charts/nos  \
	  --version 0.1.0 \
	  --namespace nebuly-nos \
	  --create-namespace

.PHONY: install-seldon-core
install-seldon-core:
	helm upgrade --install seldon-core seldon-core-operator \
	   --repo https://storage.googleapis.com/seldon-charts \
	   --set usageMetrics.enabled=true \
	   --set istio.enabled=true \
	   --namespace seldon-systeml \
	   --version $(SELDON_CORE_VERSION) \
	   --create-namespace \

.PHONY: install-kube-prometheus
install-kube-prometheus:
	helm upgrade --install nos-monitoring kube-prometheus \
    --namespace nos-monitoring \
    --create-namespace \
    --repo https://charts.bitnami.com/bitnami

.PHONY: install-nvidia-device-plugin
install-nvidia-device-plugin:
	helm upgrade --install nvdp nvdp/nvidia-device-plugin \
	  -f values/nvidia-device-plugin.yaml \
	  --namespace nvidia-device-plugin \
	  --create-namespace \
	  --version 0.13.0

.PHONY: install-gpu-operator
install-gpu-operator:
	helm upgrade --install --wait \
		-f ./values/gpu-operator.yaml \
	 	-n gpu-operator \
	 	--create-namespace \
	 	nvidia-gpu-operator \
	 	nvidia/gpu-operator \
	 	--version $(NVIDIA_GPU_OPERATOR_VERSION) \


.PHONY: install
install: install-cert-manager install-istio install-kube-prometheus install-nvidia-device-plugin install-gpu-operator install-nebuly-device-plugin install-nos install-seldon-core ## Install all components

##@ Build
.PYHONY: docker-build
docker-build: ## Build client Docker image
	docker build -t $(CLIENT_IMG) -f client/Dockerfile ./client
 
.PYHONY: docker-push
docker-push: ## Push client Docker image
	docker push $(CLIENT_IMG)

##@ Deploy 
.PHONY: deploy-mig
deploy-mig: kustomize
	cd manifests/base && $(KUSTOMIZE) edit set image cifar10-client=${CLIENT_IMG}
	$(KUSTOMIZE) build manifests/overlays/mig

.PHONY: deploy-mps
deploy-mps: kustomize
	$(KUSTOMIZE) build manifests/overlays/mps

.PHONY: deploy-ts
deploy-ts: kustomize
	$(KUSTOMIZE) build manifests/overlays/ts
