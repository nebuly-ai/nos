# Demo KServe

The goal of this demo is to deploy [kserve](https://kserve.github.io/website/0.8/) and [knative](https://knative.dev/docs/)
in a KIND cluster and use it for serving a model using [Triton](https://developer.nvidia.com/nvidia-triton-inference-server) 
inference server.

KServe is installed along Knative in order to have serverless inference deployments: when there are no requests for a
certain time, knative automatically scales to 0 the inference server deployment so that resources are freed up.

## How to run

1. Setup the KIND cluster
```shell
make setup
```

2. Install kserve, knative and istio
```shell
make install
```

3. Deploy the model 
```shell
make deploy
```

## Execute inference requests
In order to send an inference request to deployed model:

1. Enable port forwarding from port 8080 of localhost to port 80 of Istio ingress gateway service
```shell
kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80
```

2. Setup variables
 ```shell
INGRESS_HOST=127.0.0.1
INGRESS_PORT=8080
SERVICE_HOSTNAME=$(kubectl get inferenceservice torchscript-cifar10 -o jsonpath='{.status.url}' | cut -d "/" -f 3)
MODEL_NAME=cifar10
INPUT_PATH=@./input.json
```

3. Download the inference request body
```shell
curl -O https://raw.githubusercontent.com/kserve/kserve/master/docs/samples/v1beta1/triton/torchscript/input.json
```

4. Make the inference request
```shell
curl -v -H "Host: ${SERVICE_HOSTNAME}" "http://${INGRESS_HOST}:${INGRESS_PORT}/v2/models/${MODEL_NAME}/infer" -d $INPUT_PATH
```
Note that the first time you send the request it might take a few seconds for getting the inference response, since 
knative has to scale from zero pods to 1 for serving the request.  After a few minutes without receiving requests, the 
replicas will scale back to zero.

5. **[Optional]** Run multiple inference requests concurrently
```shell
hey -z 30s -c 5 -m POST -host ${SERVICE_HOSTNAME} -d $INPUT_PATH "http://${INGRESS_HOST}:${INGRESS_PORT}/v2/models/$MODEL_NAME/infer"
```
```shell
Summary:
  Total:        30.0079 secs
  Slowest:      0.0957 secs
  Fastest:      0.0047 secs
  Average:      0.0100 secs
  Requests/sec: 497.6351
  
  Total data:   1075176 bytes
  Size/request: 72 bytes

Response time histogram:
  0.005 [1]     |
  0.014 [13825] |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.023 [1045]  |■■■
  0.032 [45]    |
  0.041 [6]     |
  0.050 [2]     |
  0.059 [2]     |
  0.068 [3]     |
  0.078 [2]     |
  0.087 [0]     |
  0.096 [2]     |


Latency distribution:
  10% in 0.0074 secs
  25% in 0.0083 secs
  50% in 0.0095 secs
  75% in 0.0110 secs
  90% in 0.0130 secs
  95% in 0.0148 secs
  99% in 0.0201 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0000 secs, 0.0047 secs, 0.0957 secs
  DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0114 sec
  resp wait:    0.0100 secs, 0.0046 secs, 0.0896 secs
  resp read:    0.0000 secs, 0.0000 secs, 0.0098 secs

Status code distribution:
  [400] 14933 responses
```