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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

/*
It imports the time package that provides the functionality
to measure and display the time. The type Context is an
interface that has a method Now that returns the Time that
returns the instant it time with a nanosecond precision.
*/
type Context interface {
	logging.Log
	Now() time.Time
	GetTimeout() time.Duration
	Credentials() *auth.Credentials
	DatastoreVersion() string
	NewQueryContext(queryContext string, readonly bool) interface{}
	AdminContext() (interface{}, error)
	QueryContext() string
	QueryContextParts() []string
	GetTxContext() interface{}
	SetTxContext(t interface{})
	Readonly() bool
	SetAdvisor()
	IncRecursionCount(inc int) int
	RecursionCount() int
	StoreValue(key string, val interface{})
	RetrieveValue(key string) interface{}
	ReleaseValue(key string)
	EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool,
		profileUdfExecTrees bool, funcKey string) (value.Value, uint64, error)
	OpenStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool,
		profileUdfExecTrees bool, funcKey string) (functions.Handle, error)
	Parse(s string) (interface{}, error)
	Infer(value.Value, value.Value) (value.Value, error)
	InferKeyspace(ks interface{}, with value.Value) (value.Value, error)
	SetTracked(bool)
	IsTracked() bool
	RecordJsCU(d time.Duration, m uint64)
	SetPreserveProjectionOrder(on bool) bool
	IsAdmin() bool
	IsPrepared() bool
}

type ExecutionHandle interface {
	Type() string
	NextDocument() (value.Value, error)
}

type CurlContext interface {
	Context
	GetAllowlist() map[string]interface{}
	UrlCredentials(urlS string) *auth.Credentials
	DatastoreURL() string
	LoadX509KeyPair(certFile, keyFile string, privateKeyPassphrase []byte) (interface{}, error)
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
	Park(stop func(bool), changeCallerState bool)
	Resume(changeCallerState bool)
	ParkableEvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery bool,
		readonly bool, profileUdfExecTrees bool, funcKey string) (value.Value, uint64, error)
	ParkableOpenStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery bool,
		readonly bool, profileUdfExecTrees bool, funcKey string) (functions.Handle, error)
}

type QuotaContext interface {
	UseRequestQuota() bool
	TrackValueSize(uint64) errors.Error
	ReleaseValueSize(uint64)
	MemoryQuota() uint64
	CurrentQuotaUsage() float64
}

// Created to avoid import cycles between plan and expression package
type ParkableExecutePreparedContext interface {
	ParkableContext
	PrepareStatementExt(statement string) (interface{}, error)

	// Only accepts non-prepared statements
	ExecutePreparedExt(prepared interface{}, namedArgs map[string]value.Value, positionalArgs value.Values) (
		value.Value, uint64, error)
}

type SequenceContext interface {
	Context
	NextSequenceValue(name string) (int64, errors.Error)
	PrevSequenceValue(name string) (int64, errors.Error)
}

type FunctionsPhaseCount interface {
	Context
	AddFunctionsPhaseCount(fname string, c uint64)
}
