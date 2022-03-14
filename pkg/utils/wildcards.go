// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// MatchWildcardRegexp creates a Regular Expression from a gNMI wild-carded path
// This follows the gNMI wildcard syntax
// https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md#wildcards-in-paths
func MatchWildcardRegexp(query string, exact bool) *regexp.Regexp {
	const legalChars = `a-zA-Z0-9_:,\-\.`
	regexpQuery := strings.ReplaceAll(query, `[`, `\[`)
	regexpQuery = strings.ReplaceAll(regexpQuery, `*`, `[`+legalChars+`]*?`) // Not greedy
	regexpQuery = strings.ReplaceAll(regexpQuery, `...`, `.*`)               // greedy
	if exact {
		return regexp.MustCompile(fmt.Sprintf("^%s$", regexpQuery))
	}
	return regexp.MustCompile(fmt.Sprintf("^%s", regexpQuery))
}

// MatchWildcardChNameRegexp creates a Regular Expression from a wild-carded path
func MatchWildcardChNameRegexp(query string, exact bool) *regexp.Regexp {
	const legalChars = `a-zA-Z0-9_:,\-\.`
	regexpQuery := strings.ReplaceAll(query, `?`, `[`+legalChars+`]{1}`)     // greedy
	regexpQuery = strings.ReplaceAll(regexpQuery, `*`, `[`+legalChars+`]*?`) // Not greedy
	if exact {
		return regexp.MustCompile(fmt.Sprintf("^%s$", regexpQuery))
	}
	return regexp.MustCompile(fmt.Sprintf("^%s", regexpQuery))
}
