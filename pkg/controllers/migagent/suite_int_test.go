//go:build integration

package migagent

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/nebuly-ai/nebulnetes/pkg/api/n8s.nebuly.ai/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/pkg/test/factory"
	testmig "github.com/nebuly-ai/nebulnetes/pkg/test/gpu/mig"
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
	"sync"
	"testing"
	"time"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var (
	ctx               context.Context
	cancel            context.CancelFunc
	actuatorMigClient *testmig.MockedMigClient
	reporterMigClient *testmig.MockedMigClient
	logger            logr.Logger
)

const (
	actuatorNodeName = "actuator-test"
	reporterNodeName = "reporter-test"

	actuatorNvidiaDevicePluginPodName = "nvidia-device-plugin-actuator"
	reporterNvidiaDevicePluginPodName = "nvidia-device-plugin-reporter"
	nvidiaDevicePluginPodNamespace    = "default"
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
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
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
		MetricsBindAddress: ":8081",
	})
	Expect(err).ToNot(HaveOccurred())

	// Create nodes
	actuatorNode := factory.BuildNode(actuatorNodeName).Get()
	Expect(k8sClient.Create(ctx, &actuatorNode)).To(Succeed())
	reporterNode := factory.BuildNode(reporterNodeName).Get()
	Expect(k8sClient.Create(ctx, &reporterNode)).To(Succeed())

	// Create nvidia-device-plugin pods
	actuatorNvidiaDevicePluginPod := factory.BuildPod(nvidiaDevicePluginPodNamespace, actuatorNvidiaDevicePluginPodName).
		WithLabel("app", "nvidia-device-plugin-daemonset").
		WithNodeName(actuatorNodeName).
		WithContainer(factory.BuildContainer("test", "test").Get()).
		Get()
	reporterNvidiaDevicePluginPod := factory.BuildPod(nvidiaDevicePluginPodNamespace, reporterNvidiaDevicePluginPodName).
		WithLabel("app", "nvidia-device-plugin-daemonset").
		WithNodeName(reporterNodeName).
		WithContainer(factory.BuildContainer("test", "test").Get()).
		Get()
	Expect(k8sClient.Create(ctx, &actuatorNvidiaDevicePluginPod)).To(Succeed())
	Expect(k8sClient.Create(ctx, &reporterNvidiaDevicePluginPod)).To(Succeed())

	// Create Reporter and Actuator
	mutext := sync.Mutex{}
	actuatorMigClient = &testmig.MockedMigClient{}
	reporterMigClient = &testmig.MockedMigClient{}

	// Setup Reporter
	reporter := NewReporter(k8sClient, reporterMigClient, &mutext, 1*time.Minute)
	err = reporter.SetupWithManager(k8sManager, "MIGReporter", reporterNodeName)
	Expect(err).ToNot(HaveOccurred())

	// Setup Actuator
	actuator := NewActuator(k8sClient, actuatorMigClient, &mutext, actuatorNodeName)
	err = actuator.SetupWithManager(k8sManager, "MIGActuator")
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
