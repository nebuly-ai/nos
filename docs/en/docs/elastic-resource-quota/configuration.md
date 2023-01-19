# Configuration

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
