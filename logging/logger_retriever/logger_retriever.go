//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package logger_retriever

import (
	"reflect"

	"fmt"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/retriever/logger"
)

type RetrieverLogger struct {
	logWriter *logger.LogWriter
}

func NewRetrieverLogger(keylist []string) *RetrieverLogger {
	return &RetrieverLogger{
		logWriter: initLogger(keylist),
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
	case logging.ERROR:
		rl.Error(args)
	case logging.WARN:
		rl.Warn(args)
	case logging.INFO:
		rl.Info(args)
	case logging.DEBUG:
		rl.Debug(args)
	}
}

// change log level
func (rl *RetrieverLogger) SetLevel(level logging.Level) {
	switch level {
	case logging.ERROR:
		rl.logWriter.SetLogLevel(logger.LevelError)
	case logging.WARN:
		rl.logWriter.SetLogLevel(logger.LevelWarn)
	case logging.INFO:
		rl.logWriter.SetLogLevel(logger.LevelInfo)
	case logging.DEBUG:
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

var QueryLogger *logger.LogWriter

const (
	HTTP      = "HTTP"
	SCAN      = "SCAN"
	OPTIMIZER = "OPTIMIZER"
	PLANNER   = "PLANNER"
	PARSER    = "PARSER"
	COMPILER  = "COMPILER"
	PIPELINE  = "PIPELINE"
	ALGEBRA   = "ALGEBRA"
	DATASTORE = "DATASTORE"
)

var loggerInitialized bool

// initialize the logger
func initLogger(keylist []string) *logger.LogWriter {

	if loggerInitialized == true {
		if keylist != nil {
			QueryLogger.EnableKeys(keylist)
		}

		return QueryLogger
	}

	var err error
	QueryLogger, err = logger.NewLogger("cbq-server", logger.LevelInfo)
	if err != nil {
		fmt.Printf("Cannot create logger instance")
		return nil
	}

	// set logging to file
	QueryLogger.SetFile()
	if keylist != nil {
		QueryLogger.EnableKeys(keylist)
	} else {
		QueryLogger.EnableKeys([]string{PARSER, COMPILER, PIPELINE, ALGEBRA, DATASTORE})
	}

	loggerInitialized = true
	return QueryLogger
}
