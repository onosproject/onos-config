// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	v2 "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_FormatInput(t *testing.T) {
	jsonBody := []byte(`{
  "site": [
    {
      "site-id": "site-1",
      "small-cell":[
        {
          "small-cell-id": "sc1"
        }
      ]
    }
  ]
}`)

	expected := `{
	"input": {
		"groups":[
			"group_1",
			"group_2"
		],
		"target":"test_target",
  "site": [
    {
      "site_id": "site_1",
      "small_cell":[
        {
          "small_cell_id": "sc1"
        }
      ]
    }
  ]
}
}`

	groups := []string{"group-1", "group-2"}
	target := v2.TargetID("test-target")

	output := FormatInput(jsonBody, groups, target)

	assert.Equal(t, expected, output)
}
