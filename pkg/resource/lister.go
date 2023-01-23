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

package resource

import (
	"github.com/nebuly-ai/nos/pkg/util"
	pdrv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
	"time"
)

const (
	// PodResourcesPath is the path to the local endpoint serving the PodResources GRPC service.
	PodResourcesPath = "/var/lib/kubelet/pod-resources"
)

func NewPodResourcesListerClient(timeout time.Duration, maxMsgSize int) (pdrv1.PodResourcesListerClient, error) {
	endpoint, err := util.LocalEndpoint(PodResourcesPath, podresources.Socket)
	if err != nil {
		return nil, err
	}
	listerClient, _, err := podresources.GetV1Client(endpoint, timeout, maxMsgSize)
	return listerClient, err
}
