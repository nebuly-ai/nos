package mig

const (
	GPUModel_A100_SMX4_40GB GPUModel = "A100-SMX4-40GB"
	GPUModel_A30            GPUModel = "A30"
)

var (
	gpuModelToAllowedMigGeometries = map[GPUModel][]Geometry{
		GPUModel_A30: {
			{
				Profile4g24gb: 1,
			},
			{
				Profile2g12gb: 2,
			},
			{
				Profile2g12gb: 1,
				Profile1g6gb:  2,
			},
			{
				Profile1g6gb: 4,
			},
		},
		GPUModel_A100_SMX4_40GB: {
			{
				Profile7g40gb: 1,
			},
			{
				Profile4g20gb: 1,
				Profile2g10gb: 1,
				Profile1g5gb:  1,
			},
			{
				Profile4g20gb: 1,
				Profile1g5gb:  3,
			},
			{
				Profile3g20gb: 2,
			},
			{
				Profile3g20gb: 1,
				Profile2g10gb: 1,
				Profile1g5gb:  1,
			},
			{
				Profile3g20gb: 1,
				Profile1g5gb:  3,
			},
			{
				Profile2g10gb: 2,
				Profile3g20gb: 1,
			},
			{
				Profile2g10gb: 1,
				Profile1g5gb:  2,
				Profile3g20gb: 1,
			},
			{
				Profile2g10gb: 3,
				Profile1g5gb:  1,
			},
			{
				Profile2g10gb: 2,
				Profile1g5gb:  3,
			},
			{
				Profile2g10gb: 1,
				Profile1g5gb:  5,
			},
			{
				Profile1g5gb: 7,
			},
		},
	}
)
