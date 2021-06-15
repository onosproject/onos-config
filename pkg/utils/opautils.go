// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"fmt"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"strings"
)

// FormatInput 	add 'input' and `groups` objects to the JSON
// Also temporarily replace `-` as OPA can't handle it
func FormatInput(json []byte, groups []string) string {
	groupsStr := ""
	for i, g := range groups {
		comma := ""
		if i < len(groups)-1 {
			comma = ","
		}
		groupsStr = fmt.Sprintf("%s\t\t\t\"%s\"%s\n", groupsStr, g, comma)
	}
	jsonTreeReplace := strings.ReplaceAll(strings.ReplaceAll(string(json[2:]), "_", "^"), "-", "_")
	return fmt.Sprintf("{\n\t\"input\": {\n\t\t\"groups\":[\n%s\t\t],\n%s\n}",
		groupsStr, jsonTreeReplace)
}

// FormatOutput replace the '-' back in where they were
func FormatOutput(body []byte) (string, error) {
	bodyText := strings.ReplaceAll(strings.ReplaceAll(string(body), "_", "-"), "^", "_")
	if !strings.Contains(bodyText, "\"result\":") {
		return "", errors.NewInvalid("Unexpected body from OPA: %s", bodyText)
	}
	if strings.Contains(bodyText, "\"result\":[]") {
		return "", nil
	}
	return bodyText, nil
}
