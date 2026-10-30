// Copyright 2026-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.

//go:build go_coverage

package main

import (
	"github.com/couchbase/gocoverage"
	"github.com/couchbase/query/logging"
)

func init() {
	gocoverage.SetLogger(func(format string, args ...any) {
		logging.Infof("CodeCov "+format, args...)
	})
}
