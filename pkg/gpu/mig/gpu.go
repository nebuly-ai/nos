package mig

type Gpu interface {
	GetAvailableMigGeometries() []Geometry
}

type Geometry map[ProfileName]uint8
