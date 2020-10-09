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
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
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
	outerlevel  int32
	ksFlags     uint32
	unnests     map[string]string
}

func NewBaseKeyspace(name string, path *algebra.Path) *BaseKeyspace {
	var keyspace string

	// for expression scans we don't have a keyspace and leave it empty
	if path != nil {

		// we use the full name, except for buckets, where we look for the underlying default collection
		// this has to be done for CBO, so that we can use the same distributions for buckets and
		// default collections, when explicitly referenced
		if path.IsCollection() {
			keyspace = path.SimpleString()
		} else {
			ks, _ := datastore.GetKeyspace(path.Parts()...)

			// if we can't find it, we use a token full name
			if ks != nil {
				keyspace = ks.QualifiedName()
			} else {
				keyspace = path.SimpleString()
			}
		}
	}

	return &BaseKeyspace{
		name:     name,
		keyspace: keyspace,
	}
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
	return copyBaseKeyspaces(src, false)
}

func CopyBaseKeyspacesWithFilters(src map[string]*BaseKeyspace) map[string]*BaseKeyspace {
	return copyBaseKeyspaces(src, true)
}

func copyBaseKeyspaces(src map[string]*BaseKeyspace, copyFilter bool) map[string]*BaseKeyspace {
	dest := make(map[string]*BaseKeyspace, len(src))

	for _, kspace := range src {
		dest[kspace.name] = &BaseKeyspace{
			name:       kspace.name,
			keyspace:   kspace.keyspace,
			ksFlags:    kspace.ksFlags,
			outerlevel: kspace.outerlevel,
		}
		if len(kspace.unnests) > 0 {
			dest[kspace.name].unnests = make(map[string]string, len(kspace.unnests))
			for a, k := range kspace.unnests {
				dest[kspace.name].unnests[a] = k
			}
		}
		if copyFilter {
			if len(kspace.filters) > 0 {
				dest[kspace.name].filters = kspace.filters.Copy()
			}
			if len(kspace.joinfilters) > 0 {
				dest[kspace.name].joinfilters = kspace.joinfilters.Copy()
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

func (this *BaseKeyspace) Outerlevel() int32 {
	return this.outerlevel
}

func (this *BaseKeyspace) SetOuterlevel(outerlevel int32) {
	this.outerlevel = outerlevel
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

func GetKeyspacesAliases(baseKeyspaces map[string]*BaseKeyspace, alias string) (
	leftKeyspaces, leftAliases []string, rightKeyspace, rightAlias string) {

	leftAliases = make([]string, 0, len(baseKeyspaces))
	leftKeyspaces = make([]string, 0, len(baseKeyspaces))
	for _, kspace := range baseKeyspaces {
		if alias != "" && kspace.Name() == alias {
			rightAlias = kspace.Name()
			rightKeyspace = kspace.Keyspace()
		} else if kspace.PlanDone() {
			leftAliases = append(leftAliases, kspace.Name())
			leftKeyspaces = append(leftKeyspaces, kspace.Keyspace())
		}
	}

	return
}
