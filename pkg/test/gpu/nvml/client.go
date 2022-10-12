package nvml

type MockedNvmlClient struct {
	MigDeviceIdToGPUIndex map[string]int
	ReturnedError         error
}

func (c MockedNvmlClient) GetGpuIndex(migDeviceId string) (int, error) {
	return c.MigDeviceIdToGPUIndex[migDeviceId], c.ReturnedError
}

func (c MockedNvmlClient) DeleteMigDevice(_ string) error {
	return c.ReturnedError
}
