# Overview

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
