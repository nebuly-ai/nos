package migagent

import (
	v1 "k8s.io/api/core/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type matchingNamePredicate struct {
	Name string
}

func (p matchingNamePredicate) Create(event event.CreateEvent) bool {
	return p.Name == event.Object.GetName()
}

func (p matchingNamePredicate) Delete(event event.DeleteEvent) bool {
	return p.Name == event.Object.GetName()
}

func (p matchingNamePredicate) Update(event event.UpdateEvent) bool {
	return p.Name == event.ObjectOld.GetName()
}

func (p matchingNamePredicate) Generic(event event.GenericEvent) bool {
	return p.Name == event.Object.GetName()
}

type nodeResourcesChangedPredicate struct {
	predicate.Funcs
}

func (p nodeResourcesChangedPredicate) Update(updateEvent event.UpdateEvent) bool {
	newNode := updateEvent.ObjectNew.(*v1.Node)
	oldNode := updateEvent.ObjectOld.(*v1.Node)
	if !reflect.DeepEqual(newNode.Status.Allocatable, oldNode.Status.Allocatable) {
		return false
	}
	return reflect.DeepEqual(newNode.Status.Capacity, oldNode.Status.Capacity)
}
