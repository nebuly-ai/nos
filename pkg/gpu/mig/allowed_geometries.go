package mig

import (
	"encoding/json"
	"errors"
	"github.com/nebuly-ai/nos/pkg/gpu"
)

type AllowedMigGeometries struct {
	Models     []gpu.Model    `json:"models"`
	Geometries []gpu.Geometry `json:"allowedGeometries"`
}

func (a *AllowedMigGeometries) UnmarshalJSON(b []byte) error {
	rr := make(map[string]json.RawMessage)
	var err error
	if err = json.Unmarshal(b, &rr); err != nil {
		return err
	}

	// Unmarshal models
	models, exists := rr["models"]
	if !exists {
		return errors.New("missing field 'models'")
	}
	if err = json.Unmarshal(models, &a.Models); err != nil {
		return err
	}

	// Unmarshal geometries
	geometries, exists := rr["allowedGeometries"]
	if !exists {
		return errors.New("missing field 'allowedGeometries'")
	}
	migGeometries := make([]map[ProfileName]int, 0)
	if err = json.Unmarshal(geometries, &migGeometries); err != nil {
		return err
	}
	a.Geometries = migGeometriesToGpuGeometries(migGeometries)

	return nil
}

func migGeometriesToGpuGeometries(migGeometries []map[ProfileName]int) []gpu.Geometry {
	var res = make([]gpu.Geometry, 0)
	for _, g := range migGeometries {
		geometry := make(gpu.Geometry)
		for p, q := range g {
			geometry[p] = q
		}
		res = append(res, geometry)
	}
	return res
}

type AllowedMigGeometriesList []AllowedMigGeometries

func (a AllowedMigGeometriesList) GroupByModel() map[gpu.Model][]gpu.Geometry {
	var res = make(map[gpu.Model][]gpu.Geometry)
	for _, ag := range a {
		for _, model := range ag.Models {
			res[model] = ag.Geometries
		}
	}
	return res
}
