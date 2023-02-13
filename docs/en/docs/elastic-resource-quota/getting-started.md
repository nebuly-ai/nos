# Getting started

## Create elastic quotas

```yaml
$ kubectl apply -f -- <<EOF 
apiVersion: nos.nebuly.com/v1alpha1
kind: ElasticQuota
metadata:
  name: quota-a
  namespace: team-a
spec:
  min:
    cpu: 2
    nos.nebuly.com/gpu-memory: 16
  max:
    cpu: 10
EOF
```

The example above creates a quota for the namespace ``team-a``, guaranteeing it 2 CPUs and 16 GB of GPU memory,
and limiting the maximum number of CPUs it can use to 10. Note that:

* the ``max`` field is optional. If it is not specified, then the Elastic Quota does not enforce any upper limits on the
  amount resources that can be created in the namespace
* you can specify any valid Kubernetes resource you want in ``max`` and ``min`` fields

## Create Pods subject to Elastic Resource Quota

Unless you deployed the `nos` scheduler as the default scheduler for your cluster, you need to instruct Kubernetes
to use it for scheduling the Pods you want to be subject to Elastic Resource Quotas.

You can do that by setting the value of the `schedulerName` field of your Pods specification to `scheduler` (or to
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
