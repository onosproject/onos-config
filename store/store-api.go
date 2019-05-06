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
package store

/**
 * The Config Store API
 *
 * The following core concepts are each maintained in their own files.
 *
 * ConfigValue - the simplest element of a configuration - just a path and a value
 * ChangeValue - a ConfigValue extended with a Remove flag
 * Change - a set of ChangeValues
 * Configuration - a set of Changes related to a device
 *
 * ConfigurationStore - a way of storing Configurations in JSON files
 * ChangeStore - a way of storing Changes in JSON files
 *
 * Interfaces should be added to this file as needed
 */
