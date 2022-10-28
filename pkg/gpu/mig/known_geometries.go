package mig

var (
	A30_AllowedGeometries = []Geometry{
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
	}

	A100_SMX4_40GB_AllowedGeometries = []Geometry{
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
	}
)
