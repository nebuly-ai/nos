package controllers

import (
	"encoding/json"
)

type ModelLibraryKind string

const (
	ModelLibraryKindAzure ModelLibraryKind = "azure"
)

type ModelLibrary struct {
	Uri  string           `json:"uri"`
	Kind ModelLibraryKind `json:"kind"`
}

func NewModelLibraryFromConfig(jsonConfig string) (*ModelLibrary, error) {
	var modelLibrary ModelLibrary
	if err := json.Unmarshal([]byte(jsonConfig), &modelLibrary); err != nil {
		return nil, err
	}
	return &modelLibrary, nil
}
