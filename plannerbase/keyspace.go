//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plannerbase

import (
	"github.com/couchbase/query/expression"
)

const (
	KS_PLAN_DONE      = 1 << iota // planning is done for this keyspace
	KS_ONCLAUSE_ONLY              // use ON-clause only for planning
	KS_PRIMARY_UNNEST             // primary unnest
)

type BaseKeyspace struct {
	name        string
	keyspace    string
	filters     Filters
	joinfilters Filters
	dnfPred     expression.Expression
	origPred    expression.Expression
	onclause    expression.Expression
	ksFlags     uint32
	unnests     map[string]string
}

func NewBaseKeyspace(name, keyspace string) *BaseKeyspace {
	rv := &BaseKeyspace{
		name:     name,
		keyspace: keyspace,
	}

	return rv
}

func (this *BaseKeyspace) PlanDone() bool {
	return (this.ksFlags & KS_PLAN_DONE) != 0
}

func (this *BaseKeyspace) SetPlanDone() {
	this.ksFlags |= KS_PLAN_DONE
}

func (this *BaseKeyspace) OnclauseOnly() bool {
	return (this.ksFlags & KS_ONCLAUSE_ONLY) != 0
}

func (this *BaseKeyspace) SetOnclauseOnly() {
	this.ksFlags |= KS_ONCLAUSE_ONLY
}

func (this *BaseKeyspace) IsPrimaryUnnest() bool {
	return (this.ksFlags & KS_PRIMARY_UNNEST) != 0
}

func (this *BaseKeyspace) SetPrimaryUnnest() {
	this.ksFlags |= KS_PRIMARY_UNNEST
}

func CopyBaseKeyspaces(src map[string]*BaseKeyspace) map[string]*BaseKeyspace {
	dest := make(map[string]*BaseKeyspace, len(src))

	for _, kspace := range src {
		dest[kspace.name] = NewBaseKeyspace(kspace.name, kspace.keyspace)
		dest[kspace.name].ksFlags = kspace.ksFlags
		if len(kspace.unnests) > 0 {
			dest[kspace.name].unnests = make(map[string]string, len(kspace.unnests))
			for a, k := range kspace.unnests {
				dest[kspace.name].unnests[a] = k
			}
		}
	}

	return dest
}

func (this *BaseKeyspace) Name() string {
	return this.name
}

func (this *BaseKeyspace) Keyspace() string {
	return this.keyspace
}

func (this *BaseKeyspace) Filters() Filters {
	return this.filters
}

func (this *BaseKeyspace) JoinFilters() Filters {
	return this.joinfilters
}

func (this *BaseKeyspace) AddFilter(filter *Filter) {
	this.filters = append(this.filters, filter)
}

func (this *BaseKeyspace) AddJoinFilter(joinfilter *Filter) {
	this.joinfilters = append(this.joinfilters, joinfilter)
}

func (this *BaseKeyspace) AddFilters(filters Filters) {
	this.filters = append(this.filters, filters...)
}

func (this *BaseKeyspace) AddJoinFilters(joinfilters Filters) {
	this.joinfilters = append(this.joinfilters, joinfilters...)
}

func (this *BaseKeyspace) SetFilters(filters, joinfilters Filters) {
	this.filters = filters
	this.joinfilters = joinfilters
}

func (this *BaseKeyspace) DnfPred() expression.Expression {
	return this.dnfPred
}

func (this *BaseKeyspace) OrigPred() expression.Expression {
	return this.origPred
}

func (this *BaseKeyspace) Onclause() expression.Expression {
	return this.onclause
}

func (this *BaseKeyspace) SetPreds(dnfPred, origPred, onclause expression.Expression) {
	this.dnfPred = dnfPred
	this.origPred = origPred
	this.onclause = onclause
}

// unnests is only populated for the primary keyspace term
func (this *BaseKeyspace) AddUnnestAlias(alias, keyspace string, size int) {
	if this.unnests == nil {
		this.unnests = make(map[string]string, size)
	}
	this.unnests[alias] = keyspace
}

func (this *BaseKeyspace) GetUnnests() map[string]string {
	return this.unnests
}
