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

func (this *builder) VisitDelete(stmt *algebra.Delete) (interface{}, error) {
	this.cover = stmt
	this.node = stmt
	this.where = stmt.Where()

	this.initialIndexAdvisor(stmt)

	ksref := stmt.KeyspaceRef()
	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	this.extractPredicates(this.where, nil)

	err = this.beginMutate(keyspace, ksref, stmt.Keys(), stmt.Indexes(), stmt.Limit(), stmt.Returning() != nil)
	if err != nil {
		return nil, err
	}

	subChildren := this.subChildren
	deleteSubChildren := make([]plan.Operator, 0, 4)

	deleteSubChildren = append(deleteSubChildren, plan.NewSendDelete(keyspace, ksref.Alias(), stmt.Limit()))

	if stmt.Returning() != nil {
		deleteSubChildren = append(deleteSubChildren, plan.NewInitialProject(stmt.Returning()), plan.NewFinalProject())
	}

	if stmt.Limit() != nil {
		seqChildren := make([]plan.Operator, 0, 3)
		if len(subChildren) > 0 {
			seqChildren = append(seqChildren, plan.NewParallel(plan.NewSequence(subChildren...), this.maxParallelism))
		}
		seqChildren = append(seqChildren, plan.NewLimit(stmt.Limit()))
		seqChildren = append(seqChildren, plan.NewParallel(plan.NewSequence(deleteSubChildren...), this.maxParallelism))
		this.children = append(this.children, plan.NewSequence(seqChildren...))
	} else {
		if len(subChildren) > 0 {
			subChildren = append(subChildren, deleteSubChildren...)
		} else {
			subChildren = deleteSubChildren
		}
		this.children = append(this.children, plan.NewParallel(plan.NewSequence(subChildren...), this.maxParallelism))
	}

	if stmt.Returning() == nil {
		this.children = append(this.children, plan.NewDiscard())
	}

	return plan.NewSequence(this.children...), nil
}
