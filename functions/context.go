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
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

type Context interface {
	logging.Log
	Now() time.Time
	GetTimeout() time.Duration
	Credentials() *auth.Credentials
	IsAdmin() bool
	DatastoreVersion() string
	NewQueryContext(queryContext string, readonly bool) interface{}
	AdminContext() (interface{}, error)
	QueryContext() string
	QueryContextParts() []string
	GetTxContext() interface{}
	SetTxContext(c interface{})
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
		profileUdfExecTrees bool, funcKey string) (Handle, error)
	ParkableEvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery bool,
		readonly bool, profileUdfExecTrees bool, funcKey string) (value.Value, uint64, error)
	ParkableOpenStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery bool,
		readonly bool, profileUdfExecTrees bool, funcKey string) (Handle, error)
	Parse(s string) (interface{}, error)
	Infer(value.Value, value.Value) (value.Value, error)
	InferKeyspace(ks interface{}, with value.Value) (value.Value, error)
	SetTracked(bool)
	IsTracked() bool
	RecordJsCU(d time.Duration, m uint64)
	SetPreserveProjectionOrder(on bool) bool
	Park(stop func(bool), changeCallerState bool)
	Resume(changeCallerState bool)
	IsPrepared() bool
}

type CurlContext interface {
	Context
	GetAllowlist() map[string]interface{}
	UrlCredentials(urlS string) *auth.Credentials
	DatastoreURL() string
	LoadX509KeyPair(certFile, keyFile string, privateKeyPassphrase []byte) (interface{}, error)
}
