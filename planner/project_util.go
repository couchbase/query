//  Copyright (c) 2019 Couchbase, Inc.
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

// TODO retire
func maybeFinalProject(children []plan.Operator) []plan.Operator {

	// TODO test cluster capabilities
	// if false {
	//	children = append(children, plan.NewFinalProject())
	// }
	return children
}

func (this *builder) buildDMLProject(projection *algebra.Projection, subChildren []plan.Operator) []plan.Operator {
	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	last := subChildren[len(subChildren)-1]
	if this.useCBO && last != nil {
		cost = last.Cost()
		cardinality = last.Cardinality()
		size = last.Size()
		frCost = last.FrCost()
		if cost > 0.0 && cardinality > 0.0 && size > 0 && frCost > 0.0 {
			cost, cardinality, size, frCost = getInitialProjectCost(projection, cost, cardinality, size, frCost)
		}
	}

	subChildren = append(subChildren, plan.NewInitialProject(projection, cost, cardinality, size, frCost))

	// TODO retire
	subChildren = maybeFinalProject(subChildren)

	return subChildren
}
