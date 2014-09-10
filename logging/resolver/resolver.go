//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package resolver

import (
	"fmt"
	"os"
	"strings"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/logging"
	"github.com/couchbaselabs/query/logging/logger_golog"
)

func NewLogger(uri string) (logging.Logger, errors.Error) {
	var logger logging.Logger
	if strings.HasPrefix(uri, "golog") {
		logger = logger_golog.NewLogger(os.Stderr, logging.Info, false)
		logging.SetLogger(logger)
		return logger, nil
	}

	return nil, errors.NewError(nil, fmt.Sprintf("Invalid logger uri: %s", uri))
}
