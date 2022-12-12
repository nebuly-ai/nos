package gpu

type Model string

const (
	GPUModel_A30            Model = "A30"
	GPUModel_A100_SXM4_40GB Model = "NVIDIA-A100-40GB-SXM4"
	GPUModel_A100_PCIe_80GB Model = "NVIDIA-A100-80GB-PCIe"
)

type GPU struct {
	Model    Model
	MemoryGB int
}
