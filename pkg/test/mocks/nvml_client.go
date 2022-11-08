package mocks

import "github.com/nebuly-ai/nebulnetes/pkg/gpu"

type MockedNvmlClient struct {
	MigDeviceIdToGPUIndex map[string]int
	ReturnedError         gpu.Error
}

func (c MockedNvmlClient) GetGpuIndex(migDeviceId string) (int, gpu.Error) {
	return c.MigDeviceIdToGPUIndex[migDeviceId], c.ReturnedError
}

func (c MockedNvmlClient) DeleteMigDevice(_ string) gpu.Error {
	return c.ReturnedError
}

func (c MockedNvmlClient) CreateMigDevice(_ string, _ int) gpu.Error {
	return c.ReturnedError
}
