package transaction

import (
	configapi "github.com/onosproject/onos-api/go/onos/config/v3"
	"strings"
)

// addDeleteChildren adds all children of the intermediate path which is required to be deleted
func addDeleteChildren(index configapi.Index, changeValues map[string]configapi.PathValue, configStore map[string]configapi.PathValue) map[string]configapi.PathValue {
	// defining new changeValues map, where we will include old changeValues map and new pathValues to be cascading deleted
	var updChangeValues = make(map[string]configapi.PathValue)
	for _, changeValue := range changeValues {
		// if this pathValue has to be deleted, then we need to search for all children of this pathValue
		if changeValue.Deleted {
			for _, value := range configStore {
				if strings.HasPrefix(value.Path, changeValue.Path) && !strings.EqualFold(value.Path, changeValue.Path) {
					value.Index = index
					value.Deleted = true
					updChangeValues[value.Path] = value
				}
			}
			// overwriting itself in the store, we want the latest value (changeValue variable)
			updChangeValues[changeValue.Path] = changeValue
		} else {
			updChangeValues[changeValue.Path] = changeValue
		}
	}
	return updChangeValues
}
