package resource

import "k8s.io/api/core/v1"

type Status string

const (
	StatusUsed    Status = "used"
	StatusFree    Status = "free"
	StatusUnknown Status = "unknown"
)

type Device struct {
	// ResourceName is the name of the resource exposed to k8s
	// (e.g. nvidia.com/gpu, nvidia.com/mig-2g10gb, etc.)
	ResourceName v1.ResourceName
	// DeviceId is the actual ID of the underlying device
	// (e.g. ID of the GPU, ID of the MIG device, etc.)
	DeviceId string
	// Status represents the status of the k8s resource (e.g. free or used)
	Status Status
}
