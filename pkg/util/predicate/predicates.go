/*
 * Copyright 2023 nebuly.com.
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

package predicate

import (
	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// MatchingName
type MatchingName struct {
	Name string
}

func (p MatchingName) Create(event event.CreateEvent) bool {
	return p.Name == event.Object.GetName()
}

func (p MatchingName) Delete(event event.DeleteEvent) bool {
	return p.Name == event.Object.GetName()
}

func (p MatchingName) Update(event event.UpdateEvent) bool {
	return p.Name == event.ObjectOld.GetName()
}

func (p MatchingName) Generic(event event.GenericEvent) bool {
	return p.Name == event.Object.GetName()
}

type NodeResourcesChanged struct {
	predicate.Funcs
}

func (p NodeResourcesChanged) Update(updateEvent event.UpdateEvent) bool {
	newNode := updateEvent.ObjectNew.(*v1.Node)
	oldNode := updateEvent.ObjectOld.(*v1.Node)
	if !cmp.Equal(newNode.Status.Allocatable, oldNode.Status.Allocatable) {
		return false
	}
	return !cmp.Equal(newNode.Status.Capacity, oldNode.Status.Capacity)
}

// AnnotationsChangedPredicate
type AnnotationsChangedPredicate struct {
	predicate.Funcs
}

func (p AnnotationsChangedPredicate) Update(updateEvent event.UpdateEvent) bool {
	return !cmp.Equal(updateEvent.ObjectOld.GetAnnotations(), updateEvent.ObjectNew.GetAnnotations())
}

// ExcludeDelete
type ExcludeDelete struct {
	predicate.Funcs
}

func (p ExcludeDelete) Delete(_ event.DeleteEvent) bool {
	return false
}
