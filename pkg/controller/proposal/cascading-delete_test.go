package proposal

import (
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"gotest.tools/assert"
	"testing"
)

func Test_CascadingDeleteAlgorithm(t *testing.T) {
	// defining store here
	var store = make(map[string]*configapi.PathValue)
	// should be deleted
	store["/x/y/z"] = &configapi.PathValue{
		Path: "/x/y/z",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: true,
	}
	// should be kept
	store["/x/y/y"] = &configapi.PathValue{
		Path: "/x/y/y",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	// should be cascadingly deleted
	store["/x/y/z/w"] = &configapi.PathValue{
		Path: "/x/y/z/w",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	// should be cascadingly deleted
	store["/x/y/z/w1"] = &configapi.PathValue{
		Path: "/x/y/z/w1",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	// should be cascadingly deleted
	store["/x/y/y1"] = &configapi.PathValue{
		Path: "/x/y/y1",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	// should be deleted
	store["/x/y/y2"] = &configapi.PathValue{
		Path: "/x/y/y2",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: true,
	}
	log.Infof("store is\n%v", store)

	// defining change values here
	var changeValues = make(map[string]*configapi.PathValue)
	changeValues["/x/y/z"] = &configapi.PathValue{
		Path: "/x/y/z",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: true,
	}
	changeValues["/x/y/y1"] = &configapi.PathValue{
		Path: "/x/y/y1",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: false,
	}
	changeValues["/x/y/y2"] = &configapi.PathValue{
		Path: "/x/y/y2",
		Value: configapi.TypedValue{
			Bytes:    []byte{0xFF, 0xFF, 0xFF},
			Type:     configapi.ValueType_BYTES,
			TypeOpts: make([]int32, 0),
		},
		Deleted: true,
	}
	log.Infof("changeValues is\n%v", changeValues)

	// cascading delete algorithm
	updChangeValues := doCascadingDelete(changeValues, store)
	log.Infof("updChangeValues is\n%v", updChangeValues)
	assert.Equal(t, 5, len(updChangeValues))
	log.Infof("updChangeValues has %d PathValues to delete", len(updChangeValues))
}
