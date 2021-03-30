//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package logger_retriever

import (
	"testing"
	/*
		"github.com/couchbase/query/logging"
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
