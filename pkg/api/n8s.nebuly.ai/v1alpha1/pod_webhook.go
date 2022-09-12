package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nebuly-ai/nebulnetes/pkg/constant"
	"github.com/nebuly-ai/nebulnetes/pkg/util"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"
	"net/http"
	. "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strconv"
)

var pdlog = logf.Log.WithName("pod-resource")

//+kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,sideEffects=None,versions=v1,name=mpod.kb.io,admissionReviewVersions=v1
//+kubebuilder:object:generate=false

type GPUMemoryLabeler struct {
	client  Client
	decoder *admission.Decoder
}

func (l *GPUMemoryLabeler) InjectClient(client Client) error {
	l.client = client
	return nil
}

func (l *GPUMemoryLabeler) InjectDecoder(decoder *admission.Decoder) error {
	l.decoder = decoder
	return nil
}

func (l *GPUMemoryLabeler) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case admissionv1.Create:
		return l.handleCreate(ctx, req)
	case admissionv1.Update:
		return l.handleUpdate(ctx, req)
	default:
		return admission.Response{AdmissionResponse: admissionv1.AdmissionResponse{Allowed: true}}
	}
}

func (l *GPUMemoryLabeler) handleCreate(ctx context.Context, req admission.Request) admission.Response {
	pdlog.V(1).Info("handle mutate - CREATE")
	return l.addGPUMemoryLabelIfMissing(ctx, req)
}

func (l *GPUMemoryLabeler) handleUpdate(ctx context.Context, req admission.Request) admission.Response {
	pdlog.V(1).Info("handle mutate - UPDATE")
	return l.addGPUMemoryLabelIfMissing(ctx, req)
}

func (l *GPUMemoryLabeler) addGPUMemoryLabelIfMissing(ctx context.Context, req admission.Request) admission.Response {
	pod, err := l.extractPod(req)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	requiredGPUMemoryGB, err := computeRequiredGPUMemoryGB(pod, 16) // TODO: use memory of smallest GPU currently present instead of fixed value
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[constant.LabelGPUMemory] = strconv.FormatInt(requiredGPUMemoryGB, 10)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (l *GPUMemoryLabeler) extractPod(req admission.Request) (v1.Pod, error) {
	pod := v1.Pod{}
	err := l.decoder.Decode(req, &pod)
	if err != nil {
		return pod, err
	}
	return pod, nil
}

func computeRequiredGPUMemoryGB(pod v1.Pod, nvidiaGPUDeviceMemoryGB int64) (int64, error) {
	var totalRequiredGB int64

	requiredResources := util.ComputePodResourceRequest(pod)
	for resourceName, quantity := range requiredResources {
		if resourceName == constant.ResourceNvidiaGPU {
			totalRequiredGB += nvidiaGPUDeviceMemoryGB * quantity.Value()
			continue
		}
		if util.IsNvidiaMigDevice(resourceName) {
			migMemory, err := util.ExtractMemoryGBFromMigFormat(resourceName)
			if err != nil {
				err = fmt.Errorf("unexpected error extracting required GPU memory from resource %q: %s", resourceName, err)
				return totalRequiredGB, err
			}
			totalRequiredGB += migMemory * quantity.Value()
			continue
		}
	}

	return totalRequiredGB, nil
}
