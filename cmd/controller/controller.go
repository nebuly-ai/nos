/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/internal/controllers/elasticquota"
	configv1alpha1 "github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/config/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"os"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(configv1alpha1.AddToScheme(scheme))
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	options := ctrl.Options{
		Scheme: scheme,
	}
	controllerConfig := configv1alpha1.CustomControllerManagerConfig{}
	var err error
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&controllerConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}
	controllerConfig.FillDefaultValues()
	setupLog.Info(fmt.Sprintf("using nvidiaGPUResourceMemoryGB=%d", *controllerConfig.NvidiaGPUResourceMemoryGB))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup ElasticQuota
	elasticQuotaReconciler := elasticquota.NewElasticQuotaReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		*controllerConfig.NvidiaGPUResourceMemoryGB,
	)
	if err = elasticQuotaReconciler.SetupWithManager(mgr, constant.ElasticQuotaControllerName); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElasticQuota")
		os.Exit(1)
	}
	if err = (&v1alpha1.ElasticQuota{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ElasticQuota")
		os.Exit(1)
	}

	// Setup CompositeElasticQuota
	compositeElasticQuotaReconciler := elasticquota.NewCompositeElasticQuotaReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		*controllerConfig.NvidiaGPUResourceMemoryGB,
	)
	if err = compositeElasticQuotaReconciler.SetupWithManager(mgr, constant.CompositeElasticQuotaControllerName); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CompositeElasticQuota")
		os.Exit(1)
	}
	if err = (&v1alpha1.CompositeElasticQuota{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "CompositeElasticQuota")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
