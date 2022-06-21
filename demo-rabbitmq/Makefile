KIND_CLUSTER ?= demo-rabbitmq
SYSTEM_ARCH ?= $(shell uname -sm | sed -e "s/ /-/g" |  tr '[:upper:]' '[:lower:]')
CLUSTER_OPERATOR_VERSION ?= 1.14.0
TOPOLOGY_OPERATOR_VERSION ?= 1.7.0
LOCALBIN ?= $(shell pwd)/bin
PRODUCER_DOCKER_IMAGE ?= nebuly/rabbitmq-producer:latest
CONSUMER_DOCKER_IMAGE ?= nebuly/rabbitmq-consumer:latest

## Location to install dependencies to
$(LOCALBIN):
	mkdir -p $(LOCALBIN)


$(LOCALBIN)/.setup: $(LOCALBIN)
	kind create cluster --name $(KIND_CLUSTER) --config kind.yaml
	touch $(LOCALBIN)/.setup

$(LOCALBIN)/local-path-storage.yaml:
	curl -L "https://raw.githubusercontent.com/rancher/local-path-provisioner/master/deploy/local-path-storage.yaml" -o $@
	kubectl apply -f $@
	kubectl annotate storageclass local-path storageclass.kubernetes.io/is-default-class=true
	kubectl delete storageclass standard || true

$(LOCALBIN)/cluster-operator.yml:
	curl -L "https://github.com/rabbitmq/cluster-operator/releases/download/v$(CLUSTER_OPERATOR_VERSION)/cluster-operator.yml" -o $@
	kubectl apply -f $@

$(LOCALBIN)/cert-manager.yml:
	curl -L https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml -o $@
	kubectl apply -f $@

$(LOCALBIN)/topology-operator.yml: $(LOCALBIN)/cert-manager.yml
	curl -L "https://github.com/rabbitmq/messaging-topology-operator/releases/download/v$(TOPOLOGY_OPERATOR_VERSION)/messaging-topology-operator-with-certmanager.yaml" -o $@
	kubectl apply -f $@


# Setup KIND cluster
setup: $(LOCALBIN)/.setup

# Install operators
install: $(LOCALBIN)/local-path-storage.yaml $(LOCALBIN)/cluster-operator.yml $(LOCALBIN)/topology-operator.yml

clean:
	rm -rf $(LOCALBIN)
	kind delete cluster $(KIND_CLUSTER)


# Deploy demo components
deploy:
	kubectl apply -f manifests/
undeploy:
	kubectl delete -f manifests/



# RabbitMQ UI credentials
credentials:
	username="$(shell kubectl get secret my-cluster-default-user -o jsonpath='{.data.username}' | base64 --decode)"
	password="$(shell kubectl get secret my-cluster-default-user -o jsonpath='{.data.password}' | base64 --decode)"

build:
	docker build -f consumer/Dockerfile -t $(CONSUMER_DOCKER_IMAGE) ./consumer
	docker build -f producer/Dockerfile -t $(PRODUCER_DOCKER_IMAGE) ./producer
	kind load docker-image $(CONSUMER_DOCKER_IMAGE) --name $(KIND_CLUSTER)
	kind load docker-image $(PRODUCER_DOCKER_IMAGE) --name $(KIND_CLUSTER)