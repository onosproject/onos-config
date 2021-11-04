package configmodel

import (
	_ "github.com/golang/protobuf/proto"
	_ "github.com/openconfig/gnmi/proto/gnmi"
	_ "github.com/openconfig/goyang/pkg/yang"
	_ "github.com/openconfig/ygot/genutil"
	_ "github.com/openconfig/ygot/ygen"
	_ "github.com/openconfig/ygot/ygot"
	_ "github.com/openconfig/ygot/ytypes"

	"github.com/onosproject/onos-config/model"
	"github.com/onosproject/onos-config/model/plugin"
)

// ConfigModelPlugin defines the model plugin for {{ .Model.Name }} {{ .Model.Version }}
type ConfigModelPlugin struct{}

func (p ConfigModelPlugin) Model() configmodel.ConfigModel {
    return ConfigModel{}
}

var _ modelplugin.ConfigModelPlugin = ConfigModelPlugin{}
