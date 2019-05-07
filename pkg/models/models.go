package models

import (
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Models tracks registered YANG models
type Models struct {
	data []*pb.ModelData
}

// NewModels initializes YANG model registry and primes it with a base set of OpenConfig models
func NewModels() *Models {
	m := &Models{
		data: []*pb.ModelData{{
			Name:         "openconfig-interfaces",
			Organization: "OpenConfig working group",
			Version:      "2.0.0",
		}, {
			Name:         "openconfig-platform",
			Organization: "OpenConfig working group",
			Version:      "0.5.0",
		}, {
			Name:         "openconfig-system",
			Organization: "OpenConfig working group",
			Version:      "0.2.0",
		}}}
	return m
}

// Register inserts a new YANG model into the registry
func (r *Models) Register(m *pb.ModelData) {
	r.data = append(r.data, m)
}

// Unregister removes the specified YANG model from the registry
func (r *Models) Unregister(m *pb.ModelData) {
	for i, v := range r.data {
		if v == m {
			r.data = append(r.data[:i], r.data[i+1:]...)
		}
	}
}

// Get returns all currently registered YANG models
func (r *Models) Get() []*pb.ModelData {
	return r.data
}
