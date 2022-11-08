package nvml

import (
	"github.com/nebuly-ai/nebulnetes/pkg/gpu"
)

type Client interface {
	GetGpuIndex(migDeviceId string) (int, gpu.Error)

	DeleteMigDevice(id string) gpu.Error

	CreateMigDevices(migProfileNames []string, gpuIndex int) gpu.Error
}
