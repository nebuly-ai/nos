package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var ceqLog = logf.Log.WithName("compositeelasticquota-resource")

func (r *CompositeElasticQuota) SetupWebhookWithManager(mgr ctrl.Manager) error {
	if client == nil {
		client = mgr.GetClient()
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-n8s-nebuly-ai-v1alpha1-compositeelasticquota,mutating=false,failurePolicy=fail,sideEffects=None,groups=n8s.nebuly.ai,resources=compositeelasticquotas,verbs=create,versions=v1alpha1,name=vcompositeelasticquota.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CompositeElasticQuota{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CompositeElasticQuota) ValidateCreate() error {
	ceqLog.V(1).Info("validate create", "name", r.Name)
	// TODO: Check that the specified namespaces do not already belong to another CompositeElasticQuota
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CompositeElasticQuota) ValidateUpdate(old runtime.Object) error {
	ceqLog.V(1).Info("validate update", "name", r.Name)
	// TODO: Check that the specified namespaces do not already belong to another CompositeElasticQuota
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CompositeElasticQuota) ValidateDelete() error {
	return nil
}
