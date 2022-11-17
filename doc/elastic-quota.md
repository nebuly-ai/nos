# Elastic Resource Quota

## Overview

Elastic Resource Quota management is based on the
[Capacity Scheduling](https://github.com/kubernetes-sigs/scheduler-plugins/tree/master/pkg/capacityscheduling) scheduler
plugin, which implements
the [Capacity Scheduling Kubernetes Enhancement Proposal](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/kep/9-capacity-scheduling/README.md)
. Nebulnetes extends the former implementation by adding the following features:

* over-quota pods preemption
* CompositeElasticQuota resources for defining limits on multiple namespaces
* custom resource `n8s.nebuly.ai/gpu-memory` for setting limits on the amount of GPU memory consumed by
  a namespace
* fair sharing of over-quota resources
* optional max limits

## How to define resource limits

You can define resource limits on namespaces using two custom resources: `ElasticQuota` and `CompositeElasticQuota`.
They both work in the same way, the only difference is that the latter lets define limits on multiple
namespaces. Limits are specified through two fields:

* `min`: the minimum resources that are guaranteed to the namespace. Nebulnetes will make sure that, at any time,
  the namespace subject to the quota will always have access to **at least** these resources.
* `max`: optional field that limits the total amount of resources that can be requested by a namespace. If not
  max is not specified, then Nebulnetes does not enforce any upper limits on the resources that can be consumed by the
  Pods of the namespace.

You can find sample definitions of these resources under the [examples](../config/operator/samples) directory.

Note that `ElasticQuota` and `CompositeElasticQuota` are treated by Nebulnetes in the same way, the only difference is 
that the latter allows you to enforce limits on multiple namespaces. This means that a namespace subject to an 
ElasticQuota can borrow resources from namespaces subject to either other ElasticQuotas or CompositeElasticQuotas and,
vice-versa, the namespaces subject to a CompositeElasticQuota can borrow resources from namespaces subject to ther
ElasticQuotas and CompositeElasticQuotas.

### Constraints
The following constraints are enforced over elastic quota resources:

* you can create at most one `ElasticQuota` per namespace
* a namespace can be subject either to one `ElasticQuota` or one `CompositeElasticQuota`, but not both
* if a resource is defined both `max` and `min` fields, then its value in `max` must be greater or equal than the 
one in `min`

### How used resources are computed

When a namespace is subject to an ElasticQuota (or to a CompositeElasticQuota), Nebulnetes computes the number
of quotas consumed by that namespace by aggregating the resources requested by the Pods belonging to
that namespace, considering **only** the pods whose phase is `Running`.

Every time the amount of consumed resources changes, Nebulnetes updates the status of the respective quota object with
the new amount of used resources. You can check how many resources have been consumed by looking at the `used` field of
the
respective ElasticQuota or CompositeElasticQuota object.

### Borrowing quotas from other namespaces

If a namespace subject to an ElasticQuota (or, equivalently, to a CompositeElasticQuota) is using all the resources
guaranteed by the `min` field of its quota, it can still host new Pods by "borrowing" quotas from other namespaces
which has available resources (e.g. from namespaces subject to other ElasticQuotas where `min` resources are not
being completely used).

Pods that are scheduled by "borrowing" unused quotas from other namespaces are called **over-quota pods**.
It is important to note that these Pods can be preempted at any time to free up resources if one of the namespaces
lending the quotas claims
them back (this happens when new Pods are submitted to one of these namespaces).

You can check whether a Pod is in over-quota by checking the value of the label `n8s.nebuly.ai/capacity`, which is
automatically created by the Nebulnetes operator for every Pod created in a namespace subject to an ElasticQuota or to
a CompositeElasticQuota. The two possible values for this label are `in-quota` and `over-quota`.

You can use this label to easily find out at any time which are the over-quota pods subject to preemption risk:

```shell
kubectl get pods --all-namespaces -l n8s.nebuly.ai/capacity="over-quota"
```

#### How over-quota pods are labelled

All the Pods created within a namespace subject to an ElasticQuota (or CompositeElasticQuota)
are labelled as `in-quota` as long as the `used` resources of the quota do not exceed its `min` resources. When
this happens and news pods are created in that namespace, when they reach the running phase they are labelled as
`over-quota`.

Nebulnetes re-evaluates the over-quota status of each Pod of a namespace every time a new Pod in that
namespace changes its phase to/from "Running". With the default configuration, Nebulnetes sorts the Pods by creation
date and, if the creation timestamp is the same, by requested resources, placing first the Pods with older creation
timestamp and with fewer requested resources. After the Pods are sorted, Nebulnetes computes the aggregated requested
resources by summing the request of each pod and marks as `over-quota` all the Pods for which `used`
is greater than `min`.

Soon it will be possible to customize the order criteria used for sorting the Pods
during this process through the Nebulnetes operator config.

### Over-quota fair sharing

In order to prevent a single namespace from consuming all the over-quotas available in the cluster depriving the
other namespaces (e.g. over-quota starvation), Nebulnetes implements a fair-sharing mechanism
that guarantees that each namespace subject to an ElasticQuota has right to a part of the available over-quotas
proportional to its `min` field.

The fair-sharing mechanism does not enforce any hard limit on the amount of over-quotas Pods that a namespace can
have, but instead it enforces fair sharing by preemption. Specifically, a Pod-A subject to Elastic-quota-A can
preempt Pod-b subject to Elastic-quota-B if the following conditions are met:

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
| Elastic Quota A | n8s.nebuly.ai/gpu-memory: 40 | None |
| Elastic Quota B | n8s.nebuly.ai/gpu-memory: 10 | None |
| Elastic Quota C | n8s.nebuly.ai/gpu-memory: 30 | None |

The table below shows the quotas usage of the cluster at two different times:

| Time | Elastic Quota A | Elastic Quota B                       | Elastic Quota C | 
|------|-----------------|---------------------------------------|-----------------|
| _t1_ | Used: 40/40 GB  | Used: 40/10 GB<br/> Over-quota: 30 GB | Used: 0 GB      | 
| _t2_ | Used: 50/40 GB  | Used 30/10 GB<br/> Over-quota: 20 GB  | Used: 0 GB      |

The cluster has a total of 30 GB of memory of available over-quotas, which at time _t1_ are all being consumed by the 
pods in the namespace subject to Elastic Quota B.

At time _t2_, a new Pod is created in the namespace subject to Elastic Quota A. Even though all the quotas of the 
cluster are currently being used, the fair sharing mechanism grants to Elastic Quota A a certain amount of over-quotas
that it can use, and in order to grant these quotas Nebulnetes can preempt one or more over-quota Pods from the 
namespace subject to Elastic Quota B. 

Specifically, the following are the amounts of over-quotas guaranteed to each of the namespaces subject to the 
Elastic Quotas defined in the table above:

* guaranteed over-quota A = 40 / (40 + 10 + 30) * (0 + 0 + (30 - 0)) = 15
* guaranteed over-quota B = 10 / (40 + 10 + 30) * (0 + 0 + (30 - 0)) = 3

Assuming that all the Pods in the cluster are requesting only 10 GB of GPU memory, an over-quota Pod from 
Elastic Quota B is preempted because the following conditions are true:

* ✅ used quotas A + new Pod A <=  min quota A + guaranteed over-quota A
  * 40 + 10 <= 40 + 15
* ✅ used over-quotas B > guaranteed over-quotas
  * 30 > 3

### GPU memory limits

Both `ElasticQuota` and `CompositeElasticQuota` resources support the custom resource `n8s.nebuly.ai/gpu-memory`.
You can use this resource in the `min` and `max` fields of the elastic quotas specification to define the
minimum amount of GPU memory (expressed in GB) guaranteed to a certain namespace and its maximum limit,
respectively.

Nebulnetes automatically computes the GPU memory requested by each Pod from the GPU resources requested
by the Pod containers and enforces the limits accordingly. The amount of memory GB corresponding to the
generic resource `nvidia.com/gpu` is defined by the field `nvidiaGPUResourceMemoryGB` of the Nebulnetes Operator
configuration, which is 16 by default. For instance, using the default configuration, the value of the
resource `n8s.nebuly.ai/gpu-memory` computed from the Pod specification below is
`(10 + 16 * 2) = 42`.

```yaml
apiVersion: apps/v1
kind: Pod
metadata:
  name: nginx-deployment
spec:
  schedulerName: n8s-scheduler
  containers:
    - name: my-container
      image: my-image:0.0.1
      resources:
        limits:
          nvidia.com/mig-1g.10gb: 1
          nvidia.com/gpu: 2
```




