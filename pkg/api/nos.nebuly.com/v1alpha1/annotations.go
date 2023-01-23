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

package v1alpha1

import "fmt"

const (
	AnnotationGpuSpecPrefix   = "nos.nebuly.com/spec-gpu"
	AnnotationGpuStatusPrefix = "nos.nebuly.com/status-gpu"

	// AnnotationPartitioningPlan indicates the partitioning plan that was applied to the node.
	AnnotationPartitioningPlan = "nos.nebuly.com/spec-partitioning-plan"
	// AnnotationReportedPartitioningPlan indicates the last partitioning plan reported by the node.
	AnnotationReportedPartitioningPlan = "nos.nebuly.com/status-partitioning-plan"
)

// AnnotationGpuStatusFormat is the format of the annotation used to expose the profiles the GPUs of a node
//
// Format:
//
//	"nos.nebuly.com/status-gpu-<gpu-index>-<profile>"
//
// Example:
//
//	"nos.nebuly.com/status-gpu-0-1g.10gb-free"
var AnnotationGpuStatusFormat = fmt.Sprintf(
	"%s-%%d-%%s-%%s",
	AnnotationGpuStatusPrefix,
)

// AnnotationGpuSpecFormat is the format of the annotation used to specify the required GPU profiles
// on the GPUs of a node
//
// Format:
//
//	"nos.nebuly.com/spec-gpu-<gpu-index>-<profile>"
//
// Example:
//
//	"nos.nebuly.com/spec-gpu-0-1g.10gb"
var AnnotationGpuSpecFormat = fmt.Sprintf(
	"%s-%%d-%%s",
	AnnotationGpuSpecPrefix,
)
