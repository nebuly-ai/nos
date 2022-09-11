package v1alpha1

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var eqlog = logf.Log.WithName("elasticquota-resource")

func (r *ElasticQuota) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-n8s-nebuly-ai-v1alpha1-elasticquota,mutating=true,failurePolicy=fail,sideEffects=None,groups=n8s.nebuly.ai,resources=elasticquotas,verbs=create;update,versions=v1,name=melasticquota.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &ElasticQuota{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ElasticQuota) Default() {
	eqlog.Info("default", "name", r.Name)
	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-n8s-nebuly-ai-v1alpha1-elasticquota,mutating=false,failurePolicy=fail,sideEffects=None,groups=n8s.nebuly.ai,resources=elasticquotas,verbs=create;update,versions=v1,name=velasticquota.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ElasticQuota{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ElasticQuota) ValidateCreate() error {
	eqlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return fmt.Errorf("FOOOOOOO")
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ElasticQuota) ValidateUpdate(old runtime.Object) error {
	eqlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ElasticQuota) ValidateDelete() error {
	eqlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
