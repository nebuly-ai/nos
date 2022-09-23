package autopartitioner

import v1 "k8s.io/api/core/v1"

type ListerRegistry interface {
	UnschedulablePodLister() UnschedulablePodLister
}

type UnschedulablePodLister struct {
}

func (l UnschedulablePodLister) List(filter PodFilter) ([]v1.Pod, error) {
	return make([]v1.Pod, 0), nil
}

type PodFilter func(pod v1.Pod) bool

var All PodFilter = func(_ v1.Pod) bool {
	return true
}

var RequestingMIGResourceFilter PodFilter = func(pod v1.Pod) bool {
	// todo
	return true
}
