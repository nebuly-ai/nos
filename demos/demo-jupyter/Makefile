.PHONY: build run

build:
	docker build -t nebuly/n8s-jupyter-demo:0.0.1 .

run:
	docker run --rm -it -p 8888:8888 nebuly/n8s-jupyter-demo:0.0.1 jupyter lab --ip 0.0.0.0 --collaborative


install-ingress:
	helm upgrade --install ingress-nginx ingress-nginx \
	  --repo https://kubernetes.github.io/ingress-nginx \
	  --namespace ingress-nginx --create-namespace -f manifests/nginx-ingress-values.yaml

install-cert-manager:
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
	kubectl apply -f manifests/

install-enterprise-gateway:
	helm  upgrade --install  enterprise-gateway \
		https://github.com/jupyter-server/enterprise_gateway/releases/download/v2.6.0/jupyter_enterprise_gateway_helm-2.6.0.tgz \
   		--namespace enterprise-gateway --create-namespace \
		-f manifests/enterprise-gateway-values.yaml

install: install-ingress install-cert-manager install-enterprise-gateway
