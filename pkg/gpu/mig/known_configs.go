package mig

const (
	Model_A100_SMX4_40GB = "A100-SMX4-40GB"
	Model_A30            = "A30"
)

var (
	gpuModelToAllowedMigGeometries = map[GPUModel][]Geometry{
		Model_A30: {
			{
				profile4g24gb: 1,
			},
			{
				profile2g12gb: 2,
			},
			{
				profile2g12gb: 1,
				profile1g6gb:  2,
			},
			{
				profile1g6gb: 4,
			},
		},
		Model_A100_SMX4_40GB: {
			{
				profile7g40gb: 1,
			},
			{
				profile4g20gb: 1,
				profile2g10gb: 1,
				profile1g5gb:  1,
			},
			{
				profile4g20gb: 1,
				profile1g5gb:  3,
			},
			{
				profile3g20gb: 2,
			},
			{
				profile3g20gb: 1,
				profile2g10gb: 1,
				profile1g5gb:  1,
			},
			{
				profile3g20gb: 1,
				profile1g5gb:  3,
			},
			{
				profile2g10gb: 2,
				profile3g20gb: 1,
			},
			{
				profile2g10gb: 1,
				profile1g5gb:  2,
				profile3g20gb: 1,
			},
			{
				profile2g10gb: 3,
				profile1g5gb:  1,
			},
			{
				profile2g10gb: 2,
				profile1g5gb:  3,
			},
			{
				profile2g10gb: 1,
				profile1g5gb:  5,
			},
			{
				profile1g5gb: 7,
			},
		},
	}
)
