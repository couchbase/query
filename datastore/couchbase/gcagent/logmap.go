//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package gcagent

import (
	"strings"

	"github.com/couchbase/gocbcore/v10"
	"github.com/couchbase/query/logging"
)

type GocbcoreTransactionLogger struct {
}

func NewGocbcoreTransactionLogger() *GocbcoreTransactionLogger {
	return &GocbcoreTransactionLogger{}
}

func (tl *GocbcoreTransactionLogger) Log(level gocbcore.LogLevel, offset int, txnID, attemptID, fmt string,
	args ...interface{}) error {
	if level == gocbcore.LogInfo {
		level = gocbcore.LogDebug
	}
	return gocbcoreLogger.Log(level, offset, txnID+"/"+attemptID+" "+fmt, args...)
}

type GocbcoreLogger struct {
}

var gocbcoreLogger GocbcoreLogger

func (l GocbcoreLogger) Log(level gocbcore.LogLevel, offset int, format string,
	args ...interface{}) error {
	prefixedFormat := "(TXGOCBCORE) " + format
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
	gocbcore.SetLogRedactionLevel(gocbcore.RedactFull)
}
