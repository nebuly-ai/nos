package mig

var GPUModels = map[string]GPUModel{
	"A30": {
		Name:          "A100-SXM4",
		TotalSM:       4,
		TotalMemoryGB: 40,
		MIGProfiles: []Profile{
			{
				Name:               "1g.6gb",
				AvailableInstances: 4,
				NumSM:              1,
				MemoryGB:           6,
			},
			{
				Name:               "2g.12gb",
				AvailableInstances: 2,
				NumSM:              2,
				MemoryGB:           12,
			},
			{
				Name:               "4g.24gb",
				AvailableInstances: 1,
				NumSM:              4,
				MemoryGB:           24,
			},
		},
	},
	"A100-SXM4": {
		Name:          "A100-SXM4",
		TotalSM:       7,
		TotalMemoryGB: 40,
		MIGProfiles: []Profile{
			{
				Name:               "1g.5gb",
				AvailableInstances: 7,
				NumSM:              1,
				MemoryGB:           5,
			},
			{
				Name:               "2g.10gb",
				AvailableInstances: 3,
				NumSM:              2,
				MemoryGB:           10,
			},
			{
				Name:               "3g.20gb",
				AvailableInstances: 2,
				NumSM:              3,
				MemoryGB:           20,
			},
			{
				Name:               "4g.20gb",
				AvailableInstances: 1,
				NumSM:              4,
				MemoryGB:           20,
			},
			{
				Name:               "7g.40gb",
				AvailableInstances: 1,
				NumSM:              7,
				MemoryGB:           40,
			},
		},
	},
	"A100-SXM4-80GB": {
		Name:          "A100-SXM4-80GB",
		TotalSM:       7,
		TotalMemoryGB: 80,
		MIGProfiles: []Profile{
			{
				Name:               "1g.10gb",
				AvailableInstances: 7,
				NumSM:              1,
				MemoryGB:           10,
			},
			{
				Name:               "2g.20gb",
				AvailableInstances: 3,
				NumSM:              2,
				MemoryGB:           20,
			},
			{
				Name:               "3g.40gb",
				AvailableInstances: 2,
				NumSM:              3,
				MemoryGB:           40,
			},
			{
				Name:               "4g.40gb",
				AvailableInstances: 1,
				NumSM:              4,
				MemoryGB:           40,
			},
			{
				Name:               "7g.80gb",
				AvailableInstances: 1,
				NumSM:              7,
				MemoryGB:           80,
			},
		},
	},
}
