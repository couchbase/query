//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package functions

// This module defines context interfaces used by functions.
// Initially, it's just a copy of expression contexts, but
// it may grow to have other execution methods, or functions
// specific, as required.
// needed to avoid circular references

import (
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/value"
)

type Context interface {
	Now() time.Time
	GetTimeout() time.Duration
	AuthenticatedUsers() []string
	Credentials() *auth.Credentials
	DatastoreVersion() string
	NewQueryContext(queryContext string, readonly bool) interface{}
	QueryContext() string
	GetTxContext() interface{}
	SetTxContext(c interface{})
	Readonly() bool
	SetAdvisor()
	IncRecursionCount(inc int) int
	RecursionCount() int
	StoreValue(key string, val interface{})
	RetrieveValue(key string) interface{}
	ReleaseValue(key string)
	EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool) (value.Value, uint64, error)
	OpenStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool) (interface {
		Type() string
		Mutations() uint64
		Results() (interface{}, uint64, error)
		Complete() (uint64, error)
		NextDocument() (value.Value, error)
		Cancel()
	}, error)
	Parse(s string) (interface{}, error)
	Infer(value.Value, value.Value) (value.Value, error)
	SetTracked(bool)
	IsTracked() bool
	IsPrepared() bool
	Park(func(bool))
	Resume()
}

type CurlContext interface {
	Context
	GetWhitelist() map[string]interface{}
	UrlCredentials(urlS string) *auth.Credentials
	DatastoreURL() string
}
