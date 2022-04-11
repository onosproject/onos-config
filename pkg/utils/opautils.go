// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	v2 "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"strings"
)

// FormatInput 	add 'input' and `groups` objects to the JSON
// Also temporarily replace `-` as OPA can't handle it
func FormatInput(json []byte, groups []string, target v2.TargetID) string {
	groupsStr := ""
	for i, g := range groups {
		comma := ""
		if i < len(groups)-1 {
			comma = ","
		}
		groupsStr = fmt.Sprintf("%s\t\t\t\"%s\"%s\n", groupsStr, g, comma)
	}
	groupsStrReplace := strings.ReplaceAll(strings.ReplaceAll(groupsStr, "_", "^"), "-", "_")
	targetStrReplace := strings.ReplaceAll(strings.ReplaceAll(string(target), "_", "^"), "-", "_")
	jsonTreeReplace := strings.ReplaceAll(strings.ReplaceAll(string(json[2:]), "_", "^"), "-", "_")
	return fmt.Sprintf("{\n\t\"input\": {\n\t\t\"groups\":[\n%s\t\t],\n\t\t\"target\":\"%s\",\n%s\n}",
		groupsStrReplace, targetStrReplace, jsonTreeReplace)
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
