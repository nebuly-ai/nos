# Elastic Resource Quota

## Overview
Nebulnetes Elastic Resource Quota management is based on the 
[Capacity Scheduling](https://github.com/kubernetes-sigs/scheduler-plugins/tree/master/pkg/capacityscheduling) scheduler 
plugin, which implements the [Capacity Scheduling Kubernetes Enhancement Proposal](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/kep/9-capacity-scheduling/README.md). 

Nebulnetes extends the former implementation by adding the following features:
* over-quota pods preemption
* CompositeElasticQuota resources for defining limits on multiple namespaces
* custom resource `n8s.nebuly.ai/gpu-memory` for setting limits on the amount of GPU memory that consumed by
a namespace
* fair sharing of over-quota resources
* optional max limits

## How to define resource limits
You can define resource limits on namespaces using two custom resources: `ElasticQuota` and `CompositeElasticQuota`.
They both work in the same way, the only difference is that the latter one lets you define limits on multiple 
namespaces. Limits are specified through two fields:

* `min`: the minimum resources that are guaranteed to the namespace. Nebulnetes will make sure that, at any time, 
the namespace subject to the quota will always have access to **at least** these resources. 
* `max`: optional field that limits the total amount of resources that can be requested by a namespace. If not 
max is not specified, then Nebulnetes does not enforce any upper limits on the resources that can be consumed by the 
Pods of the namespace.

You can find sample definitions of these resources under the [examples](../config/operator/samples) directory.

### Used resources
When a namespace is subject to an ElasticQuota (or to a CompositeElasticQuota), Nebulnetes computes the amount 
of quotas consumed by that namespace by aggregating the resources requested by the Pods belonging to 
that namespace, considering **only** the pods which phase is `Running`. 

Everytime the amount of consumed resources changes, Nebulnetes updates the status of the respective quota object with 
the new amount of used resources. You can check how many resources have been consumed by looking at the `used` field of the 
respective ElasticQuota or CompositeElasticQuota object. 

### Borrowing quotas from other namespaces 
If a namespace subject to an ElasticQuota (or, equivalently, to a CompositeElasticQuota) is using all the resources 
guaranteed by the `min` field of its quota, it can still host new Pods by "borrowing" quotas from other namespaces
which has available resources (e.g. from namespaces subject to other ElasticQuotas where `min` resources are not 
being completely used).

Pods that are scheduled by "borrowing" quotas are called **over-quota pods**. It is important to note that these 
Pods can be preempted at any time in order to free up resources if one of the namespaces lending the quotas claims 
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
this happens and news pods are created in that namespace, when they reach the running status they are labelled as 
`over-quota`. 

Nebulnetes re-evaluates the "over-quota status" of each Pod of a namespace everytime a new Pod in that 
namespace changes its phase to or from "Running". With the default configuration, Nebulnetes sorts the Pods by creation 
date and, if the creation timestamp is the same, by requested resources, placing first the Pods with older creation 
timestamp and with less requested resources. After the Pods are sorted, Nebulnetes computes the aggregated requested 
resources by summing the resources requested by each pod and marking as `over-quota` all the Pods for which `used`
is greater then `min`. Soon it will be possible to configure the order criteria used for sorting the Pods 
during this process. 

### Over-quota fair sharing
TODO

### GPU memory limits
Both `ElasticQuota` and `CompositeElasticQuota` resources support the custom resource `n8s.nebuly.ai/gpu-memory`. 
You can use this resource in the `min` and `max` fields of the elastic quotas specification, for defining the 
minimum amount of GPU memory (expressed in GB) guaranteed to a certain namespace and its maximum upper limit, 
respectively.

Nebulnetes automatically computes the GPU memory requested by each Pod from the GPU resources requested
by the Pod containers, and enforces the limits accordingly. The amount of memory GB corresponding to the 
generic resource `nvidia.com/gpu` is defined by the field `nvidiaGPUResourceMemoryGB` of the Nebulnetes Operator
configuration, which is 16 by default. For instance, the `n8s.nebuly.ai/gpu-memory` resource computed from the Pod specification below is
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




