//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

type PrepareContext struct {
	requestId       string
	queryContext    string
	namedArgs       map[string]value.Value
	positionalArgs  value.Values
	indexApiVersion int
	featureControls uint64
	useFts          bool
	useCBO          bool
	optimizer       Optimizer
	deltaKeyspaces  map[string]bool
	dsContext       datastore.QueryContext
	isPrepare       bool

	// cache of ReplaceParameters() results
	replaced map[string]expression.Expression
}

func NewPrepareContext(rv *PrepareContext, requestId, queryContext string,
	namedArgs map[string]value.Value, positionalArgs value.Values,
	indexApiVersion int, featureControls uint64, useFts, useCBO bool, optimizer Optimizer,
	deltaKeyspaces map[string]bool, dsContext datastore.QueryContext, isPrepare bool) {
	rv.requestId = requestId
	rv.queryContext = queryContext
	rv.namedArgs = namedArgs
	rv.positionalArgs = positionalArgs
	rv.indexApiVersion = indexApiVersion
	rv.featureControls = featureControls
	rv.useFts = useFts
	rv.useCBO = useCBO
	rv.optimizer = optimizer
	rv.deltaKeyspaces = deltaKeyspaces
	rv.dsContext = dsContext
	rv.isPrepare = isPrepare
	return
}

func (this *PrepareContext) RequestId() string {
	return this.requestId
}

func (this *PrepareContext) QueryContext() string {
	return this.queryContext
}

func (this *PrepareContext) NamedArgs() map[string]value.Value {
	return this.namedArgs
}

func (this *PrepareContext) PositionalArgs() value.Values {
	return this.positionalArgs
}

// HasParameters returns if this context has named and/or positional parameters
func (this *PrepareContext) HasParameters() bool {
	return len(this.namedArgs) > 0 || len(this.positionalArgs) > 0
}

// ReplaceParameters replaces named/positional parameters in expr with their argument values.
// The result is cached and reused across calls with the same source expression.
//
// Pass needCopy true when the caller intends to mutate the returned expression (e.g. via SetExprFlag)
// or hand it off to something else that might mutate it
func (this *PrepareContext) ReplaceParameters(expr expression.Expression, needCopy bool) (expression.Expression, error) {
	if expr == nil || !this.HasParameters() {
		return expr, nil
	}

	if this.replaced == nil {
		this.replaced = make(map[string]expression.Expression, 8)
	}

	str := expr.String()
	var replaced expression.Expression
	if cached, ok := this.replaced[str]; ok {
		replaced = cached
	} else {
		var err error
		replaced, err = base.ReplaceParameters(expr, this.namedArgs, this.positionalArgs)
		if err != nil {
			return nil, err
		}
		this.replaced[str] = replaced
	}

	// the cached entry is shared across every caller thus the caller needs to request a copy
	// if it needs to mutate its result (or otherwise not risk affecting other holders of the
	// cached instance).
	// a copy is needed whether this call populated the cache or just hit it.
	if needCopy {
		return replaced.Copy(), nil
	}
	return replaced, nil
}

func (this *PrepareContext) IndexApiVersion() int {
	return this.indexApiVersion
}

func (this *PrepareContext) FeatureControls() uint64 {
	return this.featureControls
}

func (this *PrepareContext) UseFts() bool {
	return this.useFts
}

func (this *PrepareContext) UseCBO() bool {
	return this.useCBO
}

func (this *PrepareContext) Optimizer() Optimizer {
	return this.optimizer
}

func (this *PrepareContext) SetDeltaKeyspaces(dk map[string]bool) {
	this.deltaKeyspaces = dk
}

func (this *PrepareContext) SetNamedArgs(na map[string]value.Value) {
	this.namedArgs = na
	this.replaced = nil
}

func (this *PrepareContext) SetPositionalArgs(pa value.Values) {
	this.positionalArgs = pa
	this.replaced = nil
}

func (this *PrepareContext) DeltaKeyspaces() map[string]bool {
	return this.deltaKeyspaces
}

func (this *PrepareContext) HasDeltaKeyspace(keyspace string) bool {
	_, ok := this.deltaKeyspaces[keyspace]
	return ok
}

// some planner usage is done by internal users (eg auto reprepare), and thus it does
// not have credentials
// we don't have to filter error messages for these use cases.
func (this *PrepareContext) Credentials() *auth.Credentials {
	if this.dsContext == nil {
		return nil
	}
	return this.dsContext.Credentials()
}

// don't provide credentials for prepared statements (MB-24871)
func (this *PrepareContext) Context() datastore.QueryContext {
	if this.isPrepare {
		return nil
	}
	return this.dsContext
}

func (this *PrepareContext) SetIsPrepare() {
	this.isPrepare = true
}
