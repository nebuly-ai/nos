/*
 * Copyright 2022 Nebuly.ai
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

package timeslicing

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
	v1 "k8s.io/api/core/v1"
)

func ParseSpecAnnotation(key, value string) (gpu.SpecAnnotation[ProfileName], error) {
	return gpu.ParseSpecAnnotation(key, value, ProfileEmpty)
}

func ParseStatusAnnotation(key, value string) (gpu.StatusAnnotation[ProfileName], error) {
	return gpu.ParseStatusAnnotation(key, value, ProfileEmpty)
}

func ParseNodeAnnotations(node v1.Node) (gpu.StatusAnnotationList[ProfileName], gpu.SpecAnnotationList[ProfileName]) {
	return gpu.ParseNodeAnnotations(node, ProfileEmpty)
}
