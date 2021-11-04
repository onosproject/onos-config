package main

import (
	"github.com/onosproject/onos-config/models/{{ .Model.Name }}_{{ .Model.Version | replace "." "_" }}/model"
)

var ConfigModelPlugin configmodel.ConfigModelPlugin
