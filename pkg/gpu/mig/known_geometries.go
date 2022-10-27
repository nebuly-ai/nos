package mig

type A100_SMX4_40GB struct {
}

func (a A100_SMX4_40GB) GetAvailableMigGeometries() []Geometry {
	return []Geometry{
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
}

type A30 struct {
}

func (a A30) GetAvailableMigGeometries() []Geometry {
	return []Geometry{
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
}
