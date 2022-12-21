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

const (
	// ReplicaGpuIdSeparator is the separator used to separate the GPU ID from the replica ID in the
	// GPU devices exposed by the NVIDIA device plugin as time-slicing shared GPUs.
	ReplicaGpuIdSeparator = "::"
	// MinSliceMemoryGB is the smallest slice size that can be created on time-slicing shared GPUs.
	MinSliceMemoryGB = 1
)
