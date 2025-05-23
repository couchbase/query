//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

// Define the current server version and minimum supported version

var (
	VERSION          = "unset: build issue" // is set by correct build process
	MIN_VERSION      = "1.0.0"
	PLAN_VERSION     = 800
	MIN_PLAN_VERSION = 711 // first defined PLAN_VERSION
)
