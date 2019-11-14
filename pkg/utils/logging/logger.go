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

package logging

import (
	"flag"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	if flag.Lookup("debug") == nil {
		log, err := zap.NewProduction()
		if err != nil {
			panic(err)
		}
		logger = log.Sugar()
	} else {
		log, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		logger = log.Sugar()
	}
}

// GetLogger returns a new logger for the given subsystem name
func GetLogger(names ...string) *zap.SugaredLogger {
	log := logger
	for _, name := range names {
		log = log.Named(name)
	}
	return log
}
