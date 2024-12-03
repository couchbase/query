//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"time"
)

const (
	CONFIG   = "config.json"
	USER     = "Administrator"
	PASSWORD = "password"

	_EMAIL_FROM = "combined_testing@query-vm"

	_QUERY_PROCESS = "cbq-engine"
	_NODE_URL      = "http://localhost:8091"
	_QUERY_URL     = "http://localhost:8093"

	_ITERATION_INTERVAL = time.Second * 10

	_INIT_WAIT            = time.Second * 10 // delay to permit cluster initialisation to completed before attempting to use it
	_RETRY_WAIT           = time.Second      // dely between retries
	_RETRY_COUNT          = 15               // maximum number of retries
	_INSTANCE_RETRY_COUNT = 60               // instance set-up may take longer so this is the limit for its retries
	_WAIT_COUNT           = 120              // number of iterations to check when waiting for a URL to be available/accessible

	_CHR_SET = " _abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" // for random string content

	_MAX_PROJECTIONS_PER_KEYSPACE = 5
	_MAX_FILTERS_PER_KEYSPACE     = 8
	_MAX_OR_CLAUSES               = 3

	_MAX_RANDOM_FIELD_DEPTH = 10 // the maximum number of times calls to generate a random field can be nested

	_WAIT_INTERVAL       = time.Second
	_MIGRATION_WAIT_TIME = time.Second * 35 // 30 seconds for it to kick in plus 5 for it to complete its checks

	_DOC_SIZE_LIMIT = 20 * 1024 * 1024 // 20 MiB
)

var _FILTER_OPS = []string{"=", ">", "<", ">=", "<=", "!="}
