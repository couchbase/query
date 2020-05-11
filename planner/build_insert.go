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

func (this *builder) VisitInsert(stmt *algebra.Insert) (interface{}, error) {
	ksref := stmt.KeyspaceRef()
	ksref.SetDefaultNamespace(this.namespace)

	keyspace, err := this.getNameKeyspace(ksref)
	if err != nil {
		return nil, err
	}

	children := make([]plan.Operator, 0, 4)

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL

	if stmt.Values() != nil {
		if this.useCBO {
			cost, cardinality = getValueScanCost(stmt.Values())
		}
		children = append(children, plan.NewValueScan(stmt.Values(), cost, cardinality))
		this.maxParallelism = (len(stmt.Values()) + 64) / 64
	} else if stmt.Select() != nil {
		sel, err := stmt.Select().Accept(this)
		if err != nil {
			return nil, err
		}

		selOp := sel.(plan.Operator)
		if this.useCBO {
			cost = selOp.Cost()
			cardinality = selOp.Cardinality()
		}
		children = append(children, selOp)
	} else {
		return nil, fmt.Errorf("INSERT missing both VALUES and SELECT.")
	}

	if this.useCBO && cost > 0.0 && cardinality > 0.0 {
		cost, cardinality = getInsertCost(keyspace, stmt.Key(), stmt.Value(), stmt.Options(), nil, cost, cardinality)
	}

	insert := plan.NewSendInsert(keyspace, ksref, stmt.Key(), stmt.Value(), stmt.Options(), nil, cost, cardinality)
	subChildren := make([]plan.Operator, 0, 4)
	subChildren = append(subChildren, insert)

	if stmt.Returning() != nil {
		subChildren = this.buildDMLProject(stmt.Returning(), subChildren)
	} else {
		subChildren = append(subChildren, plan.NewDiscard(cost, cardinality))
	}

	children = append(children, this.addParallel(subChildren...))
	return plan.NewSequence(children...), nil
}
