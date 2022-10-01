//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

	subChildren = append(subChildren, plan.NewInitialProject(projection, cost, cardinality, size, frCost, true))

	// TODO retire
	subChildren = maybeFinalProject(subChildren)

	return subChildren
}
