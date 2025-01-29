//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

const (
	PLAN_COST_NOT_AVAIL = -1.0 // cost is not available
	PLAN_CARD_NOT_AVAIL = -1.0 // cardinality is not available
	PLAN_SIZE_NOT_AVAIL = -1   // document size is not available
)

func marshalOptEstimate(oe *optEstimate) map[string]interface{} {
	var r map[string]interface{}
	if oe.cost > 0.0 || oe.cardinality > 0.0 || oe.size > 0 || oe.frCost > 0.0 {
		r = make(map[string]interface{}, 4)
	}
	if oe.cost > 0.0 {
		r["cost"] = oe.cost
	}
	if oe.cardinality > 0.0 {
		r["cardinality"] = oe.cardinality
	}
	if oe.size > 0 {
		r["size"] = oe.size
	}
	if oe.frCost > 0.0 {
		r["fr_cost"] = oe.frCost
	}

	return r
}

func unmarshalOptEstimate(oe *optEstimate, unmarshalled map[string]interface{}) {
	var hasCost, hasCard, hasSize, hasFrCost bool

	if costv, ok := unmarshalled["cost"]; ok {
		if cost, ok := costv.(float64); ok && cost > 0.0 {
			oe.cost = cost
			hasCost = true
		}
	}
	if !hasCost {
		oe.cost = PLAN_COST_NOT_AVAIL
	}

	if cardinalityv, ok := unmarshalled["cardinality"]; ok {
		if cardinality, ok := cardinalityv.(float64); ok && cardinality > 0.0 {
			oe.cardinality = cardinality
			hasCard = true
		}
	}
	if !hasCard {
		oe.cardinality = PLAN_CARD_NOT_AVAIL
	}

	if sizev, ok := unmarshalled["size"]; ok {
		switch size := sizev.(type) {
		case int64:
			if size > 0 {
				oe.size = size
				hasSize = true
			}
		case float64:
			if size > 0.0 {
				oe.size = int64(size)
				hasSize = true
			}
		}
	}
	if !hasSize {
		oe.size = PLAN_SIZE_NOT_AVAIL
	}

	if frCostv, ok := unmarshalled["fr_cost"]; ok {
		if frCost, ok := frCostv.(float64); ok && frCost > 0.0 {
			oe.frCost = frCost
			hasFrCost = true
		}
	}
	if !hasFrCost {
		oe.frCost = PLAN_COST_NOT_AVAIL
	}
}
