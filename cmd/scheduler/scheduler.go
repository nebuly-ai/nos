package main

import (
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	"sigs.k8s.io/scheduler-plugins/pkg/capacityscheduling"

	_ "sigs.k8s.io/scheduler-plugins/apis/config"
	_ "sigs.k8s.io/scheduler-plugins/apis/scheduling/scheme"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewSchedulerCommand(
		app.WithPlugin(capacityscheduling.Name, capacityscheduling.New),
	)

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
