package view

import (
	"encoding/json"

	"github.com/linecard/self/pkg/convention/config"
	"github.com/linecard/self/pkg/convention/manifest"
)

type DeployTimeView struct {
	Manifest manifest.DeployTime
	Computed config.Computed
}

type BuildTimeView struct {
	Manifest manifest.BuildTime
	Computed config.Computed
}

func (d DeployTimeView) Json() (string, error) {
	j, err := json.Marshal(d)
	return string(j), err
}

func (b BuildTimeView) Json() (string, error) {
	j, err := json.Marshal(b)
	return string(j), err
}
