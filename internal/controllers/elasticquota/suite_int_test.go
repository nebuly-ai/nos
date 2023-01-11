//go:build integration

/*
 * Copyright 2023 Nebuly.ai.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package elasticquota

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nos/pkg/constant"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var (
	ctx    context.Context
	cancel context.CancelFunc
	logger logr.Logger
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controllers Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.Background())
	logger = logf.FromContext(ctx)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "operator", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// Setup ElasticQuota controller
	eqReconciler := NewElasticQuotaReconciler(
		k8sManager.GetClient(),
		k8sManager.GetScheme(),
		constant.DefaultNvidiaGPUResourceMemory,
	)
	err = eqReconciler.SetupWithManager(k8sManager, constant.ElasticQuotaControllerName)
	Expect(err).ToNot(HaveOccurred())

	// Setup CompositeElasticQuota controller
	ceqReconciler := NewCompositeElasticQuotaReconciler(
		k8sManager.GetClient(),
		k8sManager.GetScheme(),
		constant.DefaultNvidiaGPUResourceMemory,
	)
	err = ceqReconciler.SetupWithManager(k8sManager, constant.CompositeElasticQuotaControllerName)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
