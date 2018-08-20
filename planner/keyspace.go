//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
)

const (
	KS_PLAN_DONE      = 1 << iota // planning is done for this keyspace
	KS_ONCLAUSE_ONLY              // use ON-clause only for planning
	KS_PRIMARY_UNNEST             // primary unnest
)

type baseKeyspace struct {
	name        string
	filters     Filters
	joinfilters Filters
	dnfPred     expression.Expression
	origPred    expression.Expression
	onclause    expression.Expression
	ksFlags     uint32
}

func newBaseKeyspace(keyspace string) *baseKeyspace {
	rv := &baseKeyspace{
		name: keyspace,
	}

	return rv
}

func (this *baseKeyspace) PlanDone() bool {
	return (this.ksFlags & KS_PLAN_DONE) != 0
}

func (this *baseKeyspace) SetPlanDone() {
	this.ksFlags |= KS_PLAN_DONE
}

func (this *baseKeyspace) OnclauseOnly() bool {
	return (this.ksFlags & KS_ONCLAUSE_ONLY) != 0
}

func (this *baseKeyspace) SetOnclauseOnly() {
	this.ksFlags |= KS_ONCLAUSE_ONLY
}

func (this *baseKeyspace) IsPrimaryUnnest() bool {
	return (this.ksFlags & KS_PRIMARY_UNNEST) != 0
}

func (this *baseKeyspace) SetPrimaryUnnest() {
	this.ksFlags |= KS_PRIMARY_UNNEST
}

func copyBaseKeyspaces(src map[string]*baseKeyspace) map[string]*baseKeyspace {
	dest := make(map[string]*baseKeyspace, len(src))

	for _, kspace := range src {
		dest[kspace.name] = newBaseKeyspace(kspace.name)
		dest[kspace.name].ksFlags = kspace.ksFlags
	}

	return dest
}
