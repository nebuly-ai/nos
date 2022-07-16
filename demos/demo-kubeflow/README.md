# Demo KubeFlow
Simple demo on using [KubeFlow](https://www.kubeflow.org/) for developing ML pipelines.

The goal of the demo is to understand whether it is possible to:

1. Optimize an existing KubeFlow pipeline, mainly in terms of the kind of hardware where
each step is executed in pipeline runs
2. Provide and SDK or extend Nebullvm/Nebulgym libraries for providing optimization features
to users building KubeFlow pipelines.


## Main issues
1. KubeFlow APIs do not allow to retrieve Pipelines source code -> we cannot automatically optimize existing pipelines


## How to run
1. Deploy a KIND cluster and install KubeFlow on it
 
```bash
make deploy
```

2. Activate port forwarding for accessing kubeflow UI
```bash
make port-forward
```
You can then access the KubeFlow UI at http://localhost:8080/.


# What could be improved in KubeFlow?
- SDK is not very user-friendly
- When I write pipelines I have to manually specify node-affinity for selecting the kind of node on which the 
step will run -> As a Data Scientist I don't want to deal with node affinity, I just want to specify the kind 
of computation the step does, and the platform should take care of scheduling on the node that best fits it.
    - see [optimize](nebulnetes/__init__.py) function: given a KubeFlow component and a model, it should automatically
    set the NodeAffinity for that step.
