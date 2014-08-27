//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package querylog

import (
	"fmt"

	"github.com/couchbaselabs/retriever/logger"
)

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
func Init(keylist []string) *logger.LogWriter {

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
