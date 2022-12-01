package gpu

type PartitioningKind string

const (
	PartitioningKindMig         PartitioningKind = "mig"
	PartitioningKindTimeSlicing PartitioningKind = "time-slicing"
	PartitioningKindHybrid      PartitioningKind = "hybrid"
)
