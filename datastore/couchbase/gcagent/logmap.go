//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package gcagent

import (
	"strings"

	"github.com/couchbase/gocbcore/v9"
	"github.com/couchbase/query/logging"
)

type GocbcoreLogger struct {
}

var gocbcoreLogger GocbcoreLogger

func (l GocbcoreLogger) Log(level gocbcore.LogLevel, offset int, format string,
	args ...interface{}) error {
	prefixedFormat := "(GOCBCORE) " + format
	switch level {
	case gocbcore.LogError:
		logging.Errorf(prefixedFormat, args...)
	case gocbcore.LogWarn:
		logging.Warnf(prefixedFormat, args...)
	case gocbcore.LogInfo:
		// Add retry request in debug mode
		// Avoid query.log flooding and reduce contention of mutex of log write
		if strings.Contains(format, "Will retry request") {
			logging.Debugf(prefixedFormat, args...)
		} else {
			logging.Infof(prefixedFormat, args...)
		}
	case gocbcore.LogDebug:
		logging.Debugf(prefixedFormat, args...)
	default:
		logging.Tracef(prefixedFormat, args...)
	}

	return nil
}

func init() {
	gocbcore.SetLogger(gocbcoreLogger)
}
