package nvml

type Client interface {
	GetGpuIndex(migDeviceId string) (int, error)

	DeleteMigDevice(id string) error

	CreateMigDevice(migProfile string, gpuIndex int) error
}
