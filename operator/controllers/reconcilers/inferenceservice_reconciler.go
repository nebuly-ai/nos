package reconcilers

import (
	"context"
	n8sv1alpha1 "github.com/nebuly-ai/nebulnetes/api/v1alpha1"
	"github.com/nebuly-ai/nebulnetes/controllers/components"
	"github.com/nebuly-ai/nebulnetes/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InferenceServiceReconciler struct {
	components.ComponentReconcilerBase
	modelLibrary     components.ModelLibrary
	inferenceService *n8sv1alpha1.InferenceService
	instance         *n8sv1alpha1.ModelDeployment
}

func NewInferenceServiceReconciler(client client.Client,
	scheme *runtime.Scheme,
	eventRecorder record.EventRecorder,
	modelLibrary components.ModelLibrary,
	instance *n8sv1alpha1.ModelDeployment) *InferenceServiceReconciler {
	return &InferenceServiceReconciler{
		ComponentReconcilerBase: components.NewComponentReconcilerBase(client, scheme, eventRecorder),
		modelLibrary:            modelLibrary,
		inferenceService:        buildInferenceService(),
		instance:                instance,
	}
}

func buildInferenceService() *kserve.InferenceService {
	return &kserve.InferenceService{
		Spec: kserve.InferenceServiceSpec{Predictor: kserve.PredictorSpec{Triton: &kserve.TritonSpec{
			PredictorExtensionSpec: kserve.PredictorExtensionSpec{
				StorageURI:     nil,
				RuntimeVersion: utils.StringAddr("22.05-py3"),
				Container: corev1.Container{Env: []corev1.EnvVar{
					{
						Name:  "OMP_NUM_THREADS",
						Value: "1",
					},
				}},
			},
		}}},
	}
}

func (r *InferenceServiceReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	//logger := log.FromContext(ctx)

	return ctrl.Result{}, nil
}
