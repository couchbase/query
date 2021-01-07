//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
		if size, ok := sizev.(int64); ok && size > 0 {
			oe.size = size
			hasSize = true
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
