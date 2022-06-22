LOCALBIN ?= $(shell pwd)/bin
KNATIVE_VERSION ?= 1.5.0
KSERVE_VERSION ?= 0.8.0
ISTIO_VERSION ?= 1.14.1
KIND_CLUSTER ?= demo-kserve
SYSTEM_ARCH ?= $(shell uname -sm | sed -e "s/ /-/g" |  tr '[:upper:]' '[:lower:]')


## Location to install dependencies to
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

$(LOCALBIN)/.setup: $(LOCALBIN)
	kind create cluster --name $(KIND_CLUSTER) --config kind.yaml
	touch $@



# Install istioctl
$(LOCALBIN)/istioctl: $(LOCALBIN)
	curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$(ISTIO_VERSION) sh -
	mv istio-$(ISTIO_VERSION)/bin/istioctl $@
	export PATH=$(LOCALBIN)/:$PATH
	rm -r istio-$(ISTIO_VERSION)

# Install istio
.PHONY: istio
istio: $(LOCALBIN)/istioctl
	istioctl install -y



$(LOCALBIN)/knative-serving-crd.yml: $(LOCALBIN)
	curl -L https://github.com/knative/serving/releases/download/knative-v$(KNATIVE_VERSION)/serving-crds.yaml -o $@
	kubectl apply -f $@

$(LOCALBIN)/knative-serving-core.yml: $(LOCALBIN)
	curl -L https://github.com/knative/serving/releases/download/knative-v$(KNATIVE_VERSION)/serving-core.yaml -o $@
	kubectl apply -f $@

$(LOCALBIN)/knative-isito.yml: $(LOCALBIN)
	curl -L https://github.com/knative/net-istio/releases/download/knative-v$(KNATIVE_VERSION)/istio.yaml -o $@
	kubectl apply -f $@

$(LOCALBIN)/knative-isito-controller.yml: $(LOCALBIN)
	curl -L https://github.com/knative/net-istio/releases/download/knative-v$(KNATIVE_VERSION)/net-istio.yaml -o $@
	kubectl apply -f $@

$(LOCALBIN)/cert-manager.yml: $(LOCALBIN)
	curl -L https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml -o $@
	kubectl apply -f $@

$(LOCALBIN)/kserve.yml: $(LOCALBIN)
	curl -L https://github.com/kserve/kserve/releases/download/v$(KSERVE_VERSION)/kserve.yaml -o $@
	kubectl apply -f $@

$(LOCALBIN)/kserve-runtimes.yml: $(LOCALBIN)
	curl -L https://github.com/kserve/kserve/releases/download/v$(KSERVE_VERSION)/kserve-runtimes.yaml
	kubectl apply -f $@




INSTALL_TARGETS := istio \
	$(LOCALBIN)/knative-serving-crd.yml \
	$(LOCALBIN)/knative-serving-core.yml \
	$(LOCALBIN)/knative-isito.yml \
	$(LOCALBIN)/knative-isito-controller.yml \
	$(LOCALBIN)/cert-manager.yml \
	$(LOCALBIN)/kserve.yml
	$(LOCALBIN)/kserve-runtimes.yml
.PHONY: install
install: $(INSTALL_TARGETS)

.PHONY: deploy
deploy:
	kubectl apply -f manifests/

.PHONY: setup
setup: $(LOCALBIN)/.setup

clean:
	kind delete cluster --name $(KIND_CLUSTER)
	rm -rf $(LOCALBIN)



