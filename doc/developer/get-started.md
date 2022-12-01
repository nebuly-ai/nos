# Developer 

## Local development
We use [Makefile](https://makefiletutorial.com/) targets for making it easy to setup a local development environment. 
You can list all the available targets by running `make help`.

### Create a local environment
You can create a local development environment just by running:

```shell
make cluster
```

The target uses [Kind](https://kind.sigs.k8s.io/) to create a local Kubernetes cluster that uses Docker containers as 
nodes.


The Nebulnetes operator uses webhooks that require SSL certificates. You can let cert-manager create and manage
them by installing it on the cluster you have created in the previous step:
```shell
make install-cert-manager
```

### Build components
You can build the Nebulnetes components by running the `docker-build-<component-name>` targets. The targets build 
the Docker images using the default image name tagged with the version defined in the first line of
the [Makefile](../../Makefile). 

Optionally, you can override the name and the tag of the Docker image by providing them as argument to the target.

#### Build GPU Partitioner
```shell
make build-gpu-partitioner 
```
```shell
make build-gpu-partitioner GPU_PARTITIONER_IMG=custom-image:tag
```


#### Build Scheduler
```shell
make build-scheduler 
```
```shell
make build-scheduler SCHEDULER_IMG=custom-image:tag
```

#### Build Operator
```shell
make build-operator 
```
```shell
make build-operator OPERATOR_IMG=custom-image:tag
```

#### Build MIG Agent
```shell
make build-mig-agent 
```
```shell
make build-mig-agent MIG_AGENT_IMG=custom-image:tag
```

### Load Docker images into the cluster
> ⚠️ If you use the tag `latest` Kubernetes will always download the image from the registry,
> ignoring the image you loaded into the cluster. 

You can load the Docker images you have built in the previous step into the cluster by running:
```shell
kind load docker-image <image-name>:<image-tag>
```

### Install components

You can install single Nebulnetes components by running:
```shell
make deploy-<component> 
````
where `<component>` is one of the following:
- `operator`
- `gpu-partitioner`
- `scheduler`
- `mig-agent`

The targets above installs the Docker images tagged with the version defined in the first line of 
the [Makefile](../../Makefile). 

You can override the Docker image name and tag by providing it as an argument to the target:
```shell
make deploy-<component> <COMPONENT>_IMG=<your-image>
```

