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
