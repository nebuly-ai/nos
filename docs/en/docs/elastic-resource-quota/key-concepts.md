# Key concepts

## Over-quotas

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

### Example

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

## GPU memory limits

Both `ElasticQuota` and `CompositeElasticQuota` resources support the custom resource `nos.nebuly.ai/gpu-memory`.
You can use this resource in the `min` and `max` fields of the elastic quotas specification to define the
minimum amount of GPU memory (expressed in GB) guaranteed to a certain namespace and its maximum limit,
respectively.

This resource is particularly useful if you use Elastic Quotas together with
[automatic GPU partitioning](../dynamic-gpu-partitioning/overview.md), since it allows you to assign resources to different
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
