//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package resolver

import (
	"os"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/logging/logger_golog"
)

func NewLogger(uri string) (logging.Logger, errors.Error) {
	var logger logging.Logger
	switch {
	case strings.HasPrefix(uri, "golog"):
		logger = logger_golog.NewLogger(os.Stderr, logging.INFO)
		logging.SetLogger(logger)
		return logger, nil

	// these are request loggers

	case uri == "builtin":
		return &logging.TempFileLogger{}, nil
	case uri == "file":
		return &logging.FileLogger{}, nil
	case uri == "null":
		return logging.NULL_LOG, nil
	}
	return nil, errors.NewAdminInvalidURL("Logger", uri)
}

func init() {
	logger := logger_golog.NewLogger(os.Stderr, logging.INFO)
	logging.SetLogger(logger)
}
