package v1beta3

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	schedschemev1beta3 "k8s.io/kube-scheduler/config/v1beta3"
	schedconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: schedconfig.GroupName, Version: "v1beta3"}

var (
	// localSchemeBuilder and AddToScheme will stay in k8s.io/kubernetes.
	localSchemeBuilder = &schedschemev1beta3.SchemeBuilder
	// AddToScheme is a global function that registers this API group & version to a scheme
	AddToScheme = localSchemeBuilder.AddToScheme
)

// addKnownTypes registers known types to the given scheme
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&CapacitySchedulingArgs{},
	)
	return nil
}

func init() {
	localSchemeBuilder.Register(addKnownTypes)
	localSchemeBuilder.Register(RegisterDefaults)
}
