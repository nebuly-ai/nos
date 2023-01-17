# Elastic Resource Quota

## Overview

`nos` extends the Kubernetes [Resource Quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/)
by implementing
the [Capacity Scheduling KEP](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/kep/9-capacity-scheduling/README.md)
and adding more flexibility through two custom resources: `ElasticQuotas` and `CompositeElasticQuotas`.

While standard Kubernetes resource quotas allow you only to define limits on the maximum
overall resource allocation of each namespace, `nos` elastic quotas let you define two
different limits:

1. `min`: the minimum resources that are guaranteed to the namespace
2. `max`: the upper bound of the resources that the namespace can consume

In this way namespaces can borrow reserved resource quotas from other namespaces that are not using them,
as long as they do not exceed their max limit (if any) and the namespaces lending the quotas do not need them.
When a namespace claims back its reserved `min` resources, pods borrowing resources from other namespaces (e.g.
over-quota pods) are preempted to make up space.

Moreover, while the standard Kubernetes quota management computes the used quotas as the aggregation of the resources
of the resource requests specified in the Pods spec, `nos` computes the used quotas by taking into account
only running Pods in order to avoid lower resource utilization due to scheduled Pods that failed to start.

Elastic Resource Quota management is based on the
[Capacity Scheduling](https://github.com/kubernetes-sigs/scheduler-plugins/tree/master/pkg/capacityscheduling) scheduler
plugin, which also implements
the [Capacity Scheduling KEP](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/kep/9-capacity-scheduling/README.md)
. `nos` extends the former implementation by adding the following features:

* over-quota pods preemption
* `CompositeElasticQuota` resources for defining limits on multiple namespaces
* custom resource `nos.nebuly.ai/gpu-memory`
* fair sharing of over-quota resources
* optional `max` limits

## Getting started

### Create elastic quotas

```yaml
$ kubectl apply -f -- <<EOF 
apiVersion: nos.nebuly.ai/v1alpha1
kind: ElasticQuota
metadata:
  name: quota-a
  namespace: team-a
spec:
  min:
    cpu: 2
    nos.nebuly.ai/gpu-memory: 16
  max:
    cpu: 10
EOF
```

The example above creates a quota for the namespace ``team-a``, guaranteeing it 2 CPUs and 16 GB of GPU memory,
and limiting the maximum number of CPUs it can use to 10. Note that:

* the ``max`` field is optional. If it is not specified, then the Elastic Quota does not enforce any upper limits on the
  amount resources that can be created in the namespace
* you can specify any valid Kubernetes resource you want in ``max`` and ``min`` fields

### Create Pods subject to Elastic Resource Quota

Unless you deployed the `nos` scheduler as the default scheduler for your cluster, you need to instruct Kubernetes
to use it for scheduling the Pods you want to be subject to Elastic Resource Quotas.

You can do that by setting the value of the `schedulerName` field of your Pods specification to `nos-scheduler` (or to
any name you chose when installing `nos`), as shown in the example below.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  schedulerName: nos-scheduler
  containers:
    - name: nginx
      image: nginx:1.14.2
      ports:
        - containerPort: 80
```

## How to define resource quotas

You can define resource limits on namespaces using two custom resources: `ElasticQuota` and `CompositeElasticQuota`.
They both work in the same way, the only difference is that the latter defines limits on multiple
namespaces instead of on a single one. Limits are specified through two fields:

* `min`: the minimum resources that are guaranteed to the namespace. `nos` will make sure that, at any time,
  the namespace subject to the quota will always have access to **at least** these resources.
* `max`: optional field that limits the total amount of resources that can be requested by a namespace. If not
  max is not specified, then `nos` does not enforce any upper limits on the resources that can be requested by
  the namespace.

You can find sample definitions of these resources under the [samples](https://github.com/nebuly-ai/nos/tree/main/config/operator/samples) directory.

Note that `ElasticQuota` and `CompositeElasticQuota` are treated by `nos` in the same way: a
namespace subject to an `ElasticQuota` can borrow resources from namespaces subject to either other elastic quotas or
composite elastic quotas and, vice-versa, namespaces subject to a `CompositeElasticQuota` can borrow resources
from namespaces subject to either elastic quotas or composite elastic quotas.

### Constraints

The following constraints are enforced over elastic quota resources:

* you can create at most one `ElasticQuota` per namespace
* a namespace can be subject either to one `ElasticQuota` or one `CompositeElasticQuota`, but not both at the same time
* if a quota resource specifies both `max` and `min` fields, then the value of the resources specified in `max` must
  be greater or equal than the ones specified in `min`

### How used resources are computed

When a namespace is subject to an ElasticQuota (or to a CompositeElasticQuota), `nos` computes the number
of quotas consumed by that namespace by aggregating the resources requested by its pods, considering **only** the
ones whose phase is `Running`. In this way, `nos` avoid lower resource utilization due to scheduled pods that
failed to start.

Every time the amount of resources consumed by a namespace changes (e.g a Pod changes its phase to or from `Running`),
the status of the respective quota object gets updated with the new amount of used resources.

You can check how many resources have been consumed by each namespace by looking at the field `used`
of the `ElasticQuota` and `CompositeElasticQuota` objects status.

## Scheduler installation options

You can add scheduling support for Elastic Resource Quota to your cluster by choosing one of the following options.
In both cases, you also need to install the `nos operator` to manage the CRDs.

### Option 1 - Use nos-scheduler (recommended)

This is the recommended option. You can deploy the nos scheduler to your cluster either as the default scheduler
or as a second scheduler that runs alongside the default one.
In the latter case, you can use the `schedulerName` field of the Pod spec to specify which scheduler should be used.

If you installed `nos` through the Helm chart, the scheduler is deployed automatically unless you set the value
`nos-scheduler.enabled=false`.

### Option 2 - Use your k8s scheduler

Since nos Elastic Quota support is implemented as a scheduler plugin, you can compile it into your k8s scheduler
and then enable it through the kube-scheduler configuration as follows:

```yaml

apiVersion: kubescheduler.config.k8s.io/v1beta2
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: false
profiles:
  - schedulerName: default-scheduler
    plugins:
      preFilter:
        enabled:
          - name: CapacityScheduling
      postFilter:
        enabled:
          - name: CapacityScheduling
        disabled:
          - name: "*"
      reserve:
        enabled:
          - name: CapacityScheduling
    pluginConfig:
      - name: CapacityScheduling
        args:
          # Defines how much GB of memory does a nvidia.com/gpu has.
          nvidiaGpuResourceMemoryGB: 32
```

In order to compile the plugin with your scheduler, you just need to add the following line to the `main.go` file
of your scheduler:

``` go
package main

import (
 "github.com/nebuly-ai/nos/pkg/scheduler/plugins/capacityscheduling"
 "k8s.io/kubernetes/cmd/kube-scheduler/app"

 // Import plugin config
 "github.com/nebuly-ai/nos/pkg/api/scheduler"
 "github.com/nebuly-ai/nos/pkg/api/scheduler/v1beta3"

 // Ensure nos.nebuly.ai/v1alpha1 package is initialized
 _ "github.com/nebuly-ai/nos/pkg/api/nos.nebuly.ai/v1alpha1"
)

func main() {
 // - rest of your code here -

 // Add plugin config to scheme
 utilruntime.Must(scheduler.AddToScheme(scheme))
 utilruntime.Must(v1beta3.AddToScheme(scheme))

 // Add plugin to scheduler command
 command := app.NewSchedulerCommand(
  // - your other plugins here - 
  app.WithPlugin(capacityscheduling.Name, capacityscheduling.New),
 )

 // - rest of your code here -
}
```

If you choose this installation option, you don't need to deploy `nos` scheduler, so you can disable it
by setting `--set nos-scheduler.enabled=false` when installing the `nos` chart.

## Over-quotas and GPU memory limits

If a namespace subject to an `ElasticQuota` (or, equivalently, to a `CompositeElasticQuota`) is using all the resources
guaranteed by the `min` field of its quota, it can still host new pods by "borrowing" quotas from other namespaces
which has available resources (e.g. from namespaces subject to other quotas where `min` resources are not
being completely used).

> Pods that are scheduled "borrowing" unused quotas from other namespaces are called **over-quota pods**.

Over-quota pods can be preempted at any time to free up resources if any of the namespaces
lending the quotas claims back its resources.

You can check whether a Pod is in over-quota by checking the value of the label `nos.nebuly.ai/capacity`, which is
automatically created and updated by the nos operator for every Pod created in a namespace subject to
an ElasticQuota or to a CompositeElasticQuota. The two possible values for this label are `in-quota` and `over-quota`.

You can use this label to easily find out at any time which are the over-quota pods subject to preemption risk:

```shell
kubectl get pods --all-namespaces -l nos.nebuly.ai/capacity="over-quota"
```

#### How over-quota pods are labelled

All the pods created within a namespace subject to a quota are labelled as `in-quota` as long as the `used`
resources of the quota do not exceed its `min` resources. When this happens and news pods are created in that namespace,
they are labelled as `over-quota` when they reach the running phase.

`nos` re-evaluates the over-quota status of each Pod of a namespace every time a new Pod in that
namespace changes its phase to/from "Running". With the default configuration, `nos` sorts the pods by creation
date and, if the creation timestamp is the same, by requested resources, placing first the pods with older creation
timestamp and with fewer requested resources. After the pods are sorted, `nos` computes the aggregated requested
resources by summing the request of each Pod, and it marks as `over-quota` all the pods for which `used`
is greater than `min`.

> 🚧 Soon it will be possible to customize the order criteria used for sorting the pods during this process through the
> nos-operator configuration.

### Over-quota fair sharing

In order to prevent a single namespace from consuming all the over-quotas available in the cluster and starving the
others, `nos` implements a fair-sharing mechanism that guarantees that each namespace subject to an ElasticQuota
has right to a part of the available over-quotas proportional to its `min` field.

The fair-sharing mechanism does not enforce any hard limit on the amount of over-quotas pods that a namespace can
have, but instead it implements fair sharing by preemption. Specifically, a Pod-A subject to elastic-quota-A can
preempt Pod-b subject to elastic-quota-B if the following conditions are met:

1. Pod-B is in over-quota
2. `used` field of Elastic-quota-A + Pod-A request <= guaranteed over-quotas A
3. used over-quotas of Elastic-quota-B > guaranteed over-quotas B

Where:

* guaranteed over-quotas A = percentage of guaranteed over-quotas A * tot. available over-quotas
* percentage of guaranteed over-quotas A = min A / sum(min_i) * 100
* tot. available over-quotas = sum( max(0, min_i - used_i ) )

#### Example

Let's assume we have a K8s cluster with the following Elastic Quota resources:

| Elastic Quota   | Min                            | Max  |
|-----------------|--------------------------------|------|
| Elastic Quota A | nos.nebuly.ai/gpu-memory: 40 | None |
| Elastic Quota B | nos.nebuly.ai/gpu-memory: 10 | None |
| Elastic Quota C | nos.nebuly.ai/gpu-memory: 30 | None |

The table below shows the quotas usage of the cluster at two different times:

| Time | Elastic Quota A | Elastic Quota B                       | Elastic Quota C |
|------|-----------------|---------------------------------------|-----------------|
| _t1_ | Used: 40/40 GB  | Used: 40/10 GB<br/> Over-quota: 30 GB | Used: 0 GB      |
| _t2_ | Used: 50/40 GB  | Used 30/10 GB<br/> Over-quota: 20 GB  | Used: 0 GB      |

The cluster has a total of 30 GB of memory of available over-quotas, which at time _t1_ are all being consumed by the
pods in the namespace subject to Elastic Quota B.

At time _t2_, a new Pod is created in the namespace subject to Elastic Quota A. Even though all the quotas of the
cluster are currently being used, the fair sharing mechanism grants to Elastic Quota A a certain amount of over-quotas
that it can use, and in order to grant these quotas nos can preempt one or more over-quota pods from the
namespace subject to Elastic Quota B.

Specifically, the following are the amounts of over-quotas guaranteed to each of the namespaces subject to the
Elastic Quotas defined in the table above:

* guaranteed over-quota A = 40 / (40 + 10 + 30) * (0 + 0 + (30 - 0)) = 15
* guaranteed over-quota B = 10 / (40 + 10 + 30) * (0 + 0 + (30 - 0)) = 3

Assuming that all the pods in the cluster are requesting only 10 GB of GPU memory, an over-quota Pod from
Elastic Quota B is preempted because the following conditions are true:

* ✅ used quotas A + new Pod A <= min quota A + guaranteed over-quota A
  * 40 + 10 <= 40 + 15
* ✅ used over-quotas B > guaranteed over-quotas
  * 30 > 3

### GPU memory limits

Both `ElasticQuota` and `CompositeElasticQuota` resources support the custom resource `nos.nebuly.ai/gpu-memory`.
You can use this resource in the `min` and `max` fields of the elastic quotas specification to define the
minimum amount of GPU memory (expressed in GB) guaranteed to a certain namespace and its maximum limit,
respectively.

This resource is particularly useful if you use Elastic Quotas together with
[automatic GPU partitioning](dynamic-gpu-partitioning.md), since it allows you to assign resources to different
teams (e.g. namespaces) in terms of GPU memory instead of in number of GPUs, and the users can than consume
request in the same terms by claiming GPU slices with a specific amount of memory, enabling an overall fine-grained
control over the GPUs of the cluster.

`nos` automatically computes the GPU memory requested by each Pod from the GPU resources requested
by its containers and enforces the limits accordingly. The amount of memory GB corresponding to the
generic resource `nvidia.com/gpu` is defined by the field `global.nvidiaGpuResourceMemoryGB` of the
installation chart, which is `32` by default.

For instance, using the default configuration, the value of the resource `nos.nebuly.ai/gpu-memory` computed from
the Pod specification below is `10+32=42`.

```yaml
apiVersion: apps/v1
kind: Pod
metadata:
  name: nginx-deployment
spec:
  schedulerName: nos-scheduler
  containers:
    - name: my-container
      image: my-image:0.0.1
      resources:
        limits:
          nvidia.com/mig-1g.10gb: 1
          nvidia.com/gpu: 1
```

## Troubleshooting

You can check the logs of the scheduler by running the following command:

```shell
 kubectl logs -n nebuly-nos -l app.kubernetes.io/component=nos-scheduler -f
```

You can check the logs of the operator by running the following command:

```shell
 kubectl logs -n nebuly-nos -l app.kubernetes.io/component=nos-operator -f
```
