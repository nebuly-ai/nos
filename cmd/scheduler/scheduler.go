package main

import (
	"github.com/nebuly-ai/nebulnetes/pkg/api/scheduler"
	"github.com/nebuly-ai/nebulnetes/pkg/api/scheduler/v1beta3"
	"github.com/nebuly-ai/nebulnetes/pkg/scheduler/plugins/capacityscheduling"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
	kubeschedulerscheme "k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"

	// Ensure n8s.nebuly.ai/v1alpha1 package is initialized
	_ "github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	// Ensure scheduler package is initialized.
	_ "github.com/nebuly-ai/nebulnetes/pkg/api/scheduler"
)

var (
	// Re-use the in-tree Scheme.
	scheme = kubeschedulerscheme.Scheme
)

func main() {
	rand.Seed(time.Now().UnixNano())

	utilruntime.Must(scheduler.AddToScheme(scheme))
	utilruntime.Must(v1beta3.AddToScheme(scheme))

	command := app.NewSchedulerCommand(
		app.WithPlugin(capacityscheduling.Name, capacityscheduling.New),
	)

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
