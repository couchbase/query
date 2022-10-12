//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package scheduler

// This module defines context interfaces used by the scheduler.
// Initially, it's just a copy of expression contexts, but
// it may grow to have other execution methods, or the scheduler
// specific, as required.
// needed to avoid circular references

import (
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/value"
)

type Context interface {
	Now() time.Time
	DatastoreVersion() string
	QueryContext() string
	EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool) (value.Value, uint64, error)
}

type CurlContext interface {
	Context
	GetAllowlist() map[string]interface{}
	Credentials() *auth.Credentials
	UrlCredentials(urlS string) *auth.Credentials
	DatastoreURL() string
}
