//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package logger_retriever

import (
	"testing"

	/*
		"github.com/couchbaselabs/query/logging"
		"github.com/couchbaselabs/query/querylog"
	*/)

func TestRetrieverLogger(t *testing.T) {
	/*
		rl := NewRetrieverLogger(nil)

		if rl == nil {
			t.Errorf("Could not create Retriever Logger")
		}

		var logApi logging.Logger = rl

		traceId := "0x007"

		// This will be logged:
		logApi.Info(traceId, querylog.PARSER, "Info message. Hello from %s", runtime.GOOS)

		// This will be logged with no trace id and default key:
		logApi.Info("Info message")

		// This will be logged with no trace id and default key:
		logApi.Info("Info message. Hello from %s", runtime.GOOS)

		// This will not:
		logApi.Info(traceId, "testKey", "this will not be logged")

		rl.logWriter.EnableTraceLogging()

		logApi.Error(traceId, querylog.DATASTORE, "this will go to traceaction log")

		rl.logWriter.DisableTraceLogging()

		logApi.Error(traceId, querylog.DATASTORE, "this will go to the file")
	*/
}
