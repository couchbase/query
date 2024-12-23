//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
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
}

func NewPrepareContext(rv *PrepareContext, requestId, queryContext string,
	namedArgs map[string]value.Value, positionalArgs value.Values,
	indexApiVersion int, featureControls uint64, useFts, useCBO bool, optimizer Optimizer,
	deltaKeyspaces map[string]bool, dsContext datastore.QueryContext) {
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
}

func (this *PrepareContext) SetPositionalArgs(pa value.Values) {
	this.positionalArgs = pa
}

func (this *PrepareContext) DeltaKeyspaces() map[string]bool {
	return this.deltaKeyspaces
}

func (this *PrepareContext) HasDeltaKeyspace(keyspace string) bool {
	_, ok := this.deltaKeyspaces[keyspace]
	return ok
}

func (this *PrepareContext) Context() datastore.QueryContext {
	return this.dsContext
}
