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

const (
	// LabelCapacityInfo specifies the status of a Pod in regard to the ElasticQuota it belongs to
	LabelCapacityInfo = "nos.nebuly.com/capacity"
	// LabelGpuPartitioning specifies the PartitioningKind that should be performed on the GPUs of a node
	LabelGpuPartitioning = "nos.nebuly.com/gpu-partitioning"
)
