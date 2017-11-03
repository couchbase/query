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

type baseKeyspace struct {
	name        string
	filters     Filters
	joinfilters Filters
	dnfPred     expression.Expression
	origPred    expression.Expression
	plandone    bool
}

func newBaseKeyspace(keyspace string) *baseKeyspace {
	rv := &baseKeyspace{
		name: keyspace,
	}

	return rv
}

func (this *baseKeyspace) PlanDone() bool {
	return this.plandone
}

func (this *baseKeyspace) SetPlanDone() {
	this.plandone = true
}

func copyBaseKeyspaces(src map[string]*baseKeyspace) map[string]*baseKeyspace {
	dest := make(map[string]*baseKeyspace, len(src))

	for _, kspace := range src {
		dest[kspace.name] = newBaseKeyspace(kspace.name)
		dest[kspace.name].plandone = kspace.plandone
	}

	return dest
}
