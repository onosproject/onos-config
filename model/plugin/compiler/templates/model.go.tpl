package configmodel

import (
    "errors"

	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"

	"github.com/onosproject/onos-config/model"
)

const (
    modelName    configmodel.Name    = {{ .Model.Name | quote }}
    modelVersion configmodel.Version = {{ .Model.Version | quote }}
)

var modelData = []*gnmi.ModelData{
    {{- range .Model.Modules }}
	{Name: {{ .Name | quote }}, Organization: {{ .Organization | quote }}, Version: {{ .Revision | quote }}},
	{{- end }}
}

var ModelInfo = configmodel.ModelInfo{
    Name: configmodel.Name({{ .Model.Name | quote }}),
    Version: configmodel.Version({{ .Model.Version | quote }}),
}

// ConfigModel defines the config model for {{ .Model.Name }} {{ .Model.Version }}
type ConfigModel struct{}

func (m ConfigModel) Info() configmodel.ModelInfo {
    return ModelInfo
}

func (m ConfigModel) Data() []*gnmi.ModelData {
    return modelData
}

func (m ConfigModel) Schema() (map[string]*yang.Entry, error) {
	return UnzipSchema()
}

func (m ConfigModel) GetStateMode() configmodel.GetStateMode {
    return configmodel.{{ .Model.GetStateMode }}
}

func (m ConfigModel) Unmarshaler() configmodel.Unmarshaler {
    return func(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
        device := &Device{}
        vgs := ygot.ValidatedGoStruct(device)
        if err := Unmarshal([]byte(jsonTree), device); err != nil {
            return nil, err
        }
        return &vgs, nil
    }
}

func (m ConfigModel) Validator() configmodel.Validator {
    return func(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
        deviceDeref := *ygotModel
        device, ok := deviceDeref.(*Device)
        if !ok {
            return errors.New("unable to convert model")
        }
        return device.Validate()
    }
}

var _ configmodel.ConfigModel = ConfigModel{}
