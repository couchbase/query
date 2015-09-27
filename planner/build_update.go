//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitUpdate(stmt *algebra.Update) (interface{}, error) {
	this.where = stmt.Where()

	ksref := stmt.KeyspaceRef()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	err = this.beginMutate(keyspace, ksref, stmt.Keys(), stmt.Indexes(), stmt.Limit(), false)
	if err != nil {
		return nil, err
	}

	subChildren := this.subChildren
	subChildren = append(subChildren, plan.NewClone(ksref.Alias()))

	if stmt.Set() != nil {
		subChildren = append(subChildren, plan.NewSet(stmt.Set()))
	}

	if stmt.Unset() != nil {
		subChildren = append(subChildren, plan.NewUnset(stmt.Unset()))
	}

	subChildren = append(subChildren, plan.NewSendUpdate(keyspace, ksref.Alias(), stmt.Limit()))

	if stmt.Returning() != nil {
		subChildren = append(subChildren, plan.NewInitialProject(stmt.Returning()), plan.NewFinalProject())
	}

	parallel := plan.NewParallel(plan.NewSequence(subChildren...), this.maxParallelism)
	this.children = append(this.children, parallel)

	if stmt.Limit() != nil {
		this.children = append(this.children, plan.NewLimit(stmt.Limit()))
	}

	if stmt.Returning() == nil {
		this.children = append(this.children, plan.NewDiscard())
	}

	return plan.NewSequence(this.children...), nil
}
