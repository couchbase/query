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
	"reflect"

	"github.com/couchbaselabs/query/accounting"
	"github.com/couchbaselabs/query/querylog"
	"github.com/couchbaselabs/retriever/logger"
)

type RetrieverLogger struct {
	logWriter *logger.LogWriter
}

func NewRetrieverLogger(keylist []string) *RetrieverLogger {
	rl := querylog.Init(keylist)
	logger := &RetrieverLogger{
		logWriter: rl,
	}
	return logger
}

func (rl *RetrieverLogger) Error(args ...interface{}) {
	if rl.logWriter == nil {
		return
	}
	rl.logWriter.LogError(getTraceID(args), getKey(args), getFormat(args), args[3:]...)
}

func (rl *RetrieverLogger) Info(args ...interface{}) {
	if rl.logWriter == nil {
		return
	}
	rl.logWriter.LogInfo(getTraceID(args), getKey(args), getFormat(args), args[3:]...)
}

func (rl *RetrieverLogger) Warn(args ...interface{}) {
	if rl.logWriter == nil {
		return
	}
	rl.logWriter.LogWarn(getTraceID(args), getKey(args), getFormat(args), args[3:]...)
}

func (rl *RetrieverLogger) Debug(args ...interface{}) {
	if rl.logWriter == nil {
		return
	}
	rl.logWriter.LogDebug(getTraceID(args), getKey(args), getFormat(args), args[3:]...)
}

func (rl *RetrieverLogger) Log(level accounting.LogLevel, args ...interface{}) {
	switch level {
	case accounting.Error:
		rl.Error(args)
	case accounting.Warn:
		rl.Warn(args)
	case accounting.Info:
		rl.Info(args)
	case accounting.Debug:
		rl.Debug(args)
	}
}

func getStringAt(i int, a []interface{}) string {
	if i < 0 || (len(a)-1) < i {
		return ""
	}
	arg := a[i]
	if arg != nil && reflect.TypeOf(arg).Kind() == reflect.String {
		return arg.(string)
	}
	return ""
}

func getTraceID(a []interface{}) string {
	return getStringAt(0, a)
}

func getKey(a []interface{}) string {
	return getStringAt(1, a)
}

func getFormat(a []interface{}) string {
	return getStringAt(2, a)
}
