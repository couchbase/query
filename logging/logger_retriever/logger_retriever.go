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

	"github.com/couchbaselabs/query/logging"
	"github.com/couchbaselabs/query/querylog"
	"github.com/couchbaselabs/retriever/logger"
)

type RetrieverLogger struct {
	logWriter *logger.LogWriter
}

func NewRetrieverLogger(keylist []string) *RetrieverLogger {
	return &RetrieverLogger{
		logWriter: querylog.Init(keylist),
	}
}

func (rl *RetrieverLogger) Error(args ...interface{}) {
	if rl.logWriter == nil {
		return
	}
	rl.logWriter.LogError(getTraceID(args), getKey(args), getMessage(args), getMessageArgs(args)...)
}

func (rl *RetrieverLogger) Info(args ...interface{}) {
	if rl.logWriter == nil {
		return
	}
	rl.logWriter.LogInfo(getTraceID(args), getKey(args), getMessage(args), getMessageArgs(args)...)
}

func (rl *RetrieverLogger) Warn(args ...interface{}) {
	if rl.logWriter == nil {
		return
	}
	rl.logWriter.LogWarn(getTraceID(args), getKey(args), getMessage(args), getMessageArgs(args)...)
}

func (rl *RetrieverLogger) Debug(args ...interface{}) {
	if rl.logWriter == nil {
		return
	}
	rl.logWriter.LogDebug(getTraceID(args), getKey(args), getMessage(args), getMessageArgs(args)...)
}

func (rl *RetrieverLogger) Log(level logging.Level, args ...interface{}) {
	switch level {
	case logging.Error:
		rl.Error(args)
	case logging.Warn:
		rl.Warn(args)
	case logging.Info:
		rl.Info(args)
	case logging.Debug:
		rl.Debug(args)
	}
}

// change log level
func (rl *RetrieverLogger) SetLevel(level logging.Level) {
	switch level {
	case logging.Error:
		rl.logWriter.SetLogLevel(logger.LevelError)
	case logging.Warn:
		rl.logWriter.SetLogLevel(logger.LevelWarn)
	case logging.Info:
		rl.logWriter.SetLogLevel(logger.LevelInfo)
	case logging.Debug:
		rl.logWriter.SetLogLevel(logger.LevelDebug)
	}
}

// Functions for mapping Logger API arguments to the Retriever API.
// Format of Retriever API arguments: <TraceID> <Key> <Message> <Message Arg> *
// Format of Logging API arguments; 0 or more arbitrary arguments: <arg> *
// Need to map Logging API arguments to Retriever API arguments in Logger API implementation:

func getTraceID(a []interface{}) string {
	if len(a) < 3 {
		return "" // No Trace ID
	}
	return getStringAt(0, a)
}

func getKey(a []interface{}) string {
	if len(a) < 3 {
		return "" // Default Key
	}
	return getStringAt(1, a)
}

func getMessage(a []interface{}) string {
	index := 2
	if len(a) < 3 { // Message is first if no trace id or key
		index = 0
	}
	return getStringAt(index, a)
}

func getMessageArgs(a []interface{}) []interface{} {
	index := 3 // Default message args index
	nargs := len(a)
	if nargs < 2 {
		return nil // No Message args if less than two arguments
	}
	if nargs == 2 {
		index = 1
	}
	return a[index:]
}

// Helper function to get a string at index i from a slice of interface{}
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
