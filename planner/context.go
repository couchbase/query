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
	"github.com/couchbase/query/settings"
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

	planStabilityMode        settings.PlanStabilityMode
	planStabilityErrorPolicy settings.PlanStabilityErrorPolicy
}

func NewPrepareContext(rv *PrepareContext, requestId, queryContext string,
	namedArgs map[string]value.Value, positionalArgs value.Values,
	indexApiVersion int, featureControls uint64, useFts, useCBO bool, optimizer Optimizer,
	deltaKeyspaces map[string]bool, dsContext datastore.QueryContext, isPrepare bool,
	planStabilityMode settings.PlanStabilityMode, planStabilityErrorPolicy settings.PlanStabilityErrorPolicy) {
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
	rv.planStabilityMode = planStabilityMode
	rv.planStabilityErrorPolicy = planStabilityErrorPolicy
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

func (this *PrepareContext) GetPlanStabilityMode() settings.PlanStabilityMode {
	return this.planStabilityMode
}

func (this *PrepareContext) IsPlanStabilityEnabled() bool {
	return this.planStabilityMode == settings.PS_MODE_PREPARED_ONLY ||
		this.planStabilityMode == settings.PS_MODE_AD_HOC ||
		this.planStabilityMode == settings.PS_MODE_AD_HOC_READ_ONLY
}

func (this *PrepareContext) IsPlanStabilityDisabled() bool {
	return this.planStabilityMode == settings.PS_MODE_OFF
}

func (this *PrepareContext) IsPlanStabilityPreparedOnly() bool {
	return this.planStabilityMode == settings.PS_MODE_PREPARED_ONLY
}

func (this *PrepareContext) IsPlanStabilityAdHoc() bool {
	return this.planStabilityMode == settings.PS_MODE_AD_HOC
}

func (this *PrepareContext) IsPlanStabilityAdHocReadOnly() bool {
	return this.planStabilityMode == settings.PS_MODE_AD_HOC_READ_ONLY
}

func (this *PrepareContext) GetPlanStabilityErrorPolicy() settings.PlanStabilityErrorPolicy {
	return this.planStabilityErrorPolicy
}
