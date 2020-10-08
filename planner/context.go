//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
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
}

func NewPrepareContext(rv *PrepareContext, requestId, queryContext string,
	namedArgs map[string]value.Value, positionalArgs value.Values,
	indexApiVersion int, featureControls uint64, useFts, useCBO bool, optimizer Optimizer,
	deltaKeyspaces map[string]bool) {
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
