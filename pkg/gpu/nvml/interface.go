package nvml

type Client interface {
	GetGpuIndex(migDeviceId string) (int, error)

	DeleteMigDevice(id string) error
}
