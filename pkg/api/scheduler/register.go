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

package scheduler

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	schedconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"
)

var SchemeGroupVersion = schema.GroupVersion{Group: schedconfig.GroupName, Version: runtime.APIVersionInternal}

var (
	localSchemeBuilder = &schedconfig.SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
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
}
