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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

//+kubebuilder:object:root=true

type OperatorConfig struct {
	metav1.TypeMeta                        `json:",inline"`
	cfg.ControllerManagerConfigurationSpec `json:",inline"`
	NvidiaGpuResourceMemoryGB              int64 `json:"NvidiaGpuResourceMemoryGB"`
}

// +kubebuilder:object:root=true

type GpuPartitionerConfig struct {
	metav1.TypeMeta                        `json:",inline"`
	cfg.ControllerManagerConfigurationSpec `json:",inline"`
	SchedulerConfigFile                    string `json:"schedulerConfigFile"`
}

//func (g *GpuPartitionerConfig) GetSchedulerConfig() (*schedulerconfig.KubeSchedulerConfiguration, error) {
//	if g.SchedulerConfigData == nil {
//		return nil, nil
//	}
//	res, err := json.Marshal(g.SchedulerConfigData)
//	if err != nil {
//		return nil, fmt.Errorf("failed to decode scheduler config: %v", err)
//	}
//	return decodeKubeSchedulerConfig(res)
//}
//
//func decodeKubeSchedulerConfig(data []byte) (*schedulerconfig.KubeSchedulerConfiguration, error) {
//	obj, gvk, err := schedulerscheme.Codecs.UniversalDecoder().Decode(data, nil, nil)
//	if err != nil {
//		return nil, err
//	}
//	if cfgObj, ok := obj.(*schedulerconfig.KubeSchedulerConfiguration); ok {
//		return cfgObj, nil
//	}
//	return nil, fmt.Errorf("couldn't decode as KubeSchedulerConfiguration, got %s: ", gvk)
//}

func init() {
	SchemeBuilder.Register(&OperatorConfig{})
	SchemeBuilder.Register(&GpuPartitionerConfig{})
}
