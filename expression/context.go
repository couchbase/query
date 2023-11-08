//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"regexp"
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/value"
)

/*
It imports the time package that provides the functionality
to measure and display the time. The type Context is an
interface that has a method Now that returns the Time that
returns the instant it time with a nanosecond precision.
*/
type Context interface {
	Now() time.Time
	GetTimeout() time.Duration
	AuthenticatedUsers() []string
	Credentials() *auth.Credentials
	DatastoreVersion() string
	NewQueryContext(queryContext string, readonly bool) interface{}
	QueryContext() string
	GetTxContext() interface{}
	SetTxContext(t interface{})
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
}

type ExecutionHandle interface {
	Type() string
	NextDocument() (value.Value, error)
}

type CurlContext interface {
	Context
	GetWhitelist() map[string]interface{}
	UrlCredentials(urlS string) *auth.Credentials
	DatastoreURL() string
}

type InlistContext interface {
	Context
	GetInlistHash(in *In) *InlistHash
	EnableInlistHash(in *In)
	RemoveInlistHash(in *In)
}

type LikeContext interface {
	Context
	GetLikeRegex(in *Like, s string) *regexp.Regexp
	CacheLikeRegex(in *Like, s string, re *regexp.Regexp)
}

type ParkableContext interface {
	Context
	Park(func(bool))
	Resume()
}
