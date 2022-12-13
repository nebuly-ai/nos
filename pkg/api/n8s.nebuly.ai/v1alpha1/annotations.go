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

package v1alpha1

const (
	AnnotationGPUSpecPrefix = "n8s.nebuly.ai/spec-gpu"

	AnnotationGPUStatusPrefix     = "n8s.nebuly.ai/status-gpu"
	AnnotationGPUStatusFreeSuffix = "free"
	AnnotationGPUStatusUsedSuffix = "used"

	// AnnotationPartitioningPlan indicates the partitioning plan that was applied to the node.
	AnnotationPartitioningPlan = "n8s.nebuly.ai/spec-partitioning-plan"
	// AnnotationReportedPartitioningPlan indicates the last partitioning plan reported by the node.
	AnnotationReportedPartitioningPlan = "n8s.nebuly.ai/status-partitioning-plan"
)
