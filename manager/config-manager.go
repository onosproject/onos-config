/*
 * Copyright 2019-present Open Networking Foundation
 *
 * Licensed under the Apache License, Configuration 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package main

import (
	"fmt"
	"onos-config/store"
)

func main() {
	var samplePath = "/test1:cont1a/cont2a/leaf2a"

	// Now create the ConfigValue
	configValue1a := store.ConfigValue{
		Path: samplePath,
		Value: "23456",
	}

	// Convert it back in to string
	fmt.Println("Welcome to config manager")
	fmt.Println(configValue1a)

	// Create ConfigValue from strings
	configValue2a, _ := store.CreateChangeValue("/test1:cont1a/cont2a/leaf2a", "13", false)
	fmt.Println(configValue2a)
}
