package instancemap

// The image below should be overwritten or directly passed by the user
// using the yaml config file for DeepLearning applications
var DEFAULT_GPU_OS string = "ami-023d5aa1ee956059c"
var DEFAULT_CPU_OS string = "ami-03ededff12e34e59e"

type InstanceInfo struct {
	name string
	cpu  string
	gpu  string
	ram  string
}

func NewInstanceInfo(name, cpu, gpu, ram string) *InstanceInfo {
	newInstance := InstanceInfo{
		cpu: cpu,
		gpu: gpu,
		ram: ram,
	}
	return &newInstance
}

func (ii InstanceInfo) HasCpu(cpuName string) bool {
	if cpuName == ii.cpu {
		return true
	} else {
		return false
	}
}

func (ii InstanceInfo) HasGpu(gpuName string) bool {
	if gpuName == ii.gpu {
		return true
	} else {
		return false
	}
}

func (ii InstanceInfo) HasRam(ramQuantity string) bool {
	if ramQuantity == ii.ram {
		return true
	}
	return false
}

type Gpu2InstanceMap struct {
	gpuName   string
	instances map[string]InstanceInfo
}

func (m *Gpu2InstanceMap) GetInstances() *map[string]InstanceInfo {
	return &m.instances
}

// TODO: Add support for other GPUs
func NewGpu2InstanceMap(gpuName string) *Gpu2InstanceMap {
	if gpuName == "NvidiaT4" {
		im := Gpu2InstanceMap{
			gpuName: gpuName,
			instances: map[string]InstanceInfo{
				"g4dn.xlarge":   {cpu: "IntelXeon", gpu: gpuName, ram: "16 GB"},
				"g4dn.2xlarge":  {cpu: "IntelXeon", gpu: gpuName, ram: "32 GB"},
				"g4dn.4xlarge":  {cpu: "IntelXeon", gpu: gpuName, ram: "64 GB"},
				"g4dn.8xlarge":  {cpu: "IntelXeon", gpu: gpuName, ram: "128 GB"},
				"g4dn.16xlarge": {cpu: "IntelXeon", gpu: gpuName, ram: "256 GB"},
				"g4ad.xlarge":   {cpu: "IntelXeon", gpu: gpuName, ram: "16 GB"},
				"g4ad.2xlarge":  {cpu: "IntelXeon", gpu: gpuName, ram: "32 GB"},
				"g4ad.4xlarge":  {cpu: "IntelXeon", gpu: gpuName, ram: "64 GB"},
			},
		}
		return &im
	} else {
		return nil
	}

}

type Cpu2InstanceMap struct {
	cpuName   string
	instances map[string]InstanceInfo
}

func (m *Cpu2InstanceMap) GetInstances() *map[string]InstanceInfo {
	return &m.instances
}

// TODO: ADD support for other CPUs
func NewCpu2InstanceMap(cpuName string) *Cpu2InstanceMap {
	if cpuName == "IntelXeon" {
		im := Cpu2InstanceMap{
			cpuName: cpuName,
			instances: map[string]InstanceInfo{
				"t3.small":   {cpu: "IntelXeon", gpu: "", ram: "2 GB"},
				"t3.medium":  {cpu: "IntelXeon", gpu: "", ram: "4 GB"},
				"t3.large":   {cpu: "IntelXeon", gpu: "", ram: "8 GB"},
				"t3.xlarge":  {cpu: "IntelXeon", gpu: "", ram: "16 GB"},
				"t3.2xlarge": {cpu: "IntelXeon", gpu: "", ram: "32 GB"},
			},
		}
		return &im
	} else {
		// return nil when the cpuName is not supported
		return nil
	}
}
