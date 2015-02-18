//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package logger_golog

import (
	"fmt"
	"os"
	"testing"

	"github.com/couchbase/query/logging"
)

func TestStub(t *testing.T) {
	logger := NewLogger(os.Stdout, logging.Debug, false)
	logging.SetLogger(logger)

	logger.Infof("This is a message from %s", "test")
	logging.Infof("This is a message from %s", "test")
	logger.Infop("This is a message from ", logging.Pair{"name", "test"}, logging.Pair{"Queue Size", 10}, logging.Pair{"Debug Mode", false})
	logging.Infop("This is a message from ", logging.Pair{"name", "test"})

	logger.Infom("This is a message from ", logging.Map{"name": "test", "Queue Size": 10, "Debug Mode": false})
	logging.Infom("This is a message from ", logging.Map{"name": "test"})

	logger.Requestf(logging.Warn, "This is a Request from %s", "test")
	logging.Requestf(logging.Info, "This is a Request from %s", "test")
	logger.Requestp(logging.Debug, "This is a Request from ", logging.Pair{"name", "test"})
	logging.Requestp(logging.Error, "This is a Request from ", logging.Pair{"name", "test"})

	logger.SetLevel(logging.Warn)
	fmt.Printf("Log level is %s\n", logger.Level())

	logger.Requestf(logging.Warn, "This is a Request from %s", "test")
	logging.Requestf(logging.Info, "This is a Request from %s", "test")
	logger.Requestp(logging.Debug, "This is a Request from ", logging.Pair{"name", "test"})
	logging.Requestp(logging.Error, "This is a Request from ", logging.Pair{"name", "test"})

	logger.Warnf("This is a message from %s", "test")
	logging.Infof("This is a message from %s", "test")
	logger.Debugp("This is a message from ", logging.Pair{"name", "test"})
	logging.Errorp("This is a message from ", logging.Pair{"name", "test"})

	fmt.Printf("Changing to json formatter\n")
	logger.entryFormatter = &jsonFormatter{}
	logger.SetLevel(logging.Debug)

	logger.Infof("This is a message from %s", "test")
	logging.Infof("This is a message from %s", "test")
	logger.Infop("This is a message from ", logging.Pair{"name", "test"}, logging.Pair{"Queue Size", 10}, logging.Pair{"Debug Mode", false})
	logging.Infop("This is a message from ", logging.Pair{"name", "test"})

	logger.Infom("This is a message from ", logging.Map{"name": "test", "Queue Size": 10, "Debug Mode": false})
	logging.Infom("This is a message from ", logging.Map{"name": "test"})

	logger.Requestf(logging.Warn, "This is a Request from %s", "test")
	logging.Requestf(logging.Info, "This is a Request from %s", "test")
	logger.Requestp(logging.Debug, "This is a Request from ", logging.Pair{"name", "test"})
	logging.Requestp(logging.Error, "This is a Request from ", logging.Pair{"name", "test"})
}
