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
)

func marshalOptEstimate(oe *optEstimate) map[string]float64 {
	if oe.cost <= 0.0 && oe.cardinality <= 0.0 {
		return nil
	}
	r := make(map[string]float64, 2)
	if oe.cost > 0.0 {
		r["cost"] = oe.cost
	}
	if oe.cardinality > 0.0 {
		r["cardinality"] = oe.cardinality
	}

	return r
}

func unmarshalOptEstimate(oe *optEstimate, unmarshalled map[string]float64) {
	if len(unmarshalled) > 0 && unmarshalled["cost"] > 0.0 {
		oe.cost = unmarshalled["cost"]
	} else {
		oe.cost = PLAN_COST_NOT_AVAIL
	}
	if len(unmarshalled) > 0 && unmarshalled["cardinality"] > 0.0 {
		oe.cardinality = unmarshalled["cardinality"]
	} else {
		oe.cardinality = PLAN_CARD_NOT_AVAIL
	}
}
