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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitUpsert(stmt *algebra.Upsert) (interface{}, error) {
	ksref := stmt.KeyspaceRef()
	ksref.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getNameKeyspace(ksref.Namespace(), ksref.Keyspace())
	if err != nil {
		return nil, err
	}

	children := make([]plan.Operator, 0, 4)

	if stmt.Values() != nil {
		children = append(children, plan.NewValueScan(stmt.Values()))
		this.maxParallelism = (len(stmt.Values()) + 64) / 64
	} else if stmt.Select() != nil {
		sel, err := stmt.Select().Accept(this)
		if err != nil {
			return nil, err
		}

		children = append(children, sel.(plan.Operator))
	} else {
		return nil, fmt.Errorf("UPSERT missing both VALUES and SELECT.")
	}

	subChildren := make([]plan.Operator, 0, 4)
	subChildren = append(subChildren, plan.NewSendUpsert(keyspace, ksref.Alias(), stmt.Key(), stmt.Value()))

	if stmt.Returning() != nil {
		subChildren = append(subChildren, plan.NewInitialProject(stmt.Returning()), plan.NewFinalProject())
	} else {
		subChildren = append(subChildren, plan.NewDiscard())
	}

	parallel := plan.NewParallel(plan.NewSequence(subChildren...), this.maxParallelism)
	children = append(children, parallel)
	return plan.NewSequence(children...), nil
}
