//go:build integration

/*
 * Copyright 2023 nebuly.com
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

package gpupartitioner_test

import (
	"context"
	"github.com/nebuly-ai/nos/internal/controllers/gpupartitioner"
	"github.com/nebuly-ai/nos/internal/partitioning/core"
	"github.com/nebuly-ai/nos/internal/partitioning/state"
	"github.com/nebuly-ai/nos/pkg/api/nos.nebuly.com/v1alpha1"
	partitioningmock "github.com/nebuly-ai/nos/pkg/test/mocks/partitioning"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/scheduler/framework"
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
	ctx                context.Context
	cancel             context.CancelFunc
	migNodeInitializer *partitioningmock.NodeInitializer
	clusterState       *state.ClusterState
)

var _ core.NodeInitializer = migNodeInitializer

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	migNodeInitializer = partitioningmock.NewNodeInitializer(t)
	RunSpecs(t, "Controllers Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.Background())

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
		Scheme:             scheme.Scheme,
		MetricsBindAddress: ":8083",
	})
	Expect(err).ToNot(HaveOccurred())

	// Init Cluster State
	clusterState = state.NewClusterState(map[string]framework.NodeInfo{})

	// Setup Node Controller
	reporter := gpupartitioner.NewNodeController(k8sClient, scheme.Scheme, migNodeInitializer, clusterState)
	Expect(reporter.SetupWithManager(k8sManager, "NodeController")).To(Succeed())

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
