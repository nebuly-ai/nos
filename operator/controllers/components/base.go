package components

import (
	"context"
	"github.com/nebuly-ai/nebulnetes/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ComponentReconciler interface {
	Reconcile(ctx context.Context) (ctrl.Result, error)
	Next(ctx context.Context) (ctrl.Result, error)
	setNext(reconciler ComponentReconciler)
}

// ComponentReconcilerBase is a base struct from inherited from all the component reconcilers. It has common fields
// and defines utility methods shared among all the reconcilers.
type ComponentReconcilerBase struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	next     ComponentReconciler
}

func NewComponentReconcilerBase(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) ComponentReconcilerBase {
	return ComponentReconcilerBase{
		client:   client,
		scheme:   scheme,
		recorder: recorder,
	}
}

func NewComponentReconcilerChain(reconcilers ...ComponentReconciler) ComponentReconciler {
	for i := 0; i < len(reconcilers)-1; i++ {
		reconcilers[i].setNext(reconcilers[i+1])
	}
	return reconcilers[0]
}

func (r *ComponentReconcilerBase) Next(ctx context.Context) (ctrl.Result, error) {
	if r.next != nil {
		return r.next.Reconcile(ctx)
	}
	return ctrl.Result{}, nil
}

func (r *ComponentReconcilerBase) setNext(reconciler ComponentReconciler) {
	r.next = reconciler
}

func (r *ComponentReconcilerBase) GetClient() client.Client {
	return r.client
}

func (r *ComponentReconcilerBase) GetScheme() *runtime.Scheme {
	return r.scheme
}

func (r *ComponentReconcilerBase) GetRecorder() record.EventRecorder {
	return r.recorder
}

func (r *ComponentReconcilerBase) HandleError(instance client.Object, err error) (ctrl.Result, error) {
	r.GetRecorder().Event(instance, corev1.EventTypeWarning, constants.EventInternalError, err.Error())
	return ctrl.Result{}, err
}

// DeleteResourceIfExists deletes the resource corresponding to the object provided as argument.
// If the resource does not exist the func does not return any error.
func (r *ComponentReconcilerBase) DeleteResourceIfExists(context context.Context, obj client.Object) error {
	logger := log.FromContext(context)

	var propagationPolicy = metav1.DeletePropagationForeground
	deleteOptions := &client.DeleteOptions{PropagationPolicy: &propagationPolicy}
	err := r.GetClient().Delete(context, obj, deleteOptions)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "unable to delete object ", "object", obj)
		return err
	}

	return nil
}

// CreateResourceIfNotExists creates the resource if it does not exist yet, setting as owner the object provided
// as the "owner" param. If the resource already exist, no error is returned.
func (r *ComponentReconcilerBase) CreateResourceIfNotExists(context context.Context, owner client.Object, obj client.Object) error {
	logger := log.FromContext(context)

	if err := controllerutil.SetControllerReference(owner, obj, r.GetScheme()); err != nil {
		logger.Error(err, "unable to set controller reference", "object", obj, "owner", owner)
		return err
	}

	err := r.GetClient().Create(context, obj)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Error(err, "unable to create object ", "object", obj)
		return err
	}

	return nil
}
