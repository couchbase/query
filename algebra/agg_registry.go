//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

func GetAggregate(name string, distinct bool) (Aggregate, bool) {
	if distinct {
		rv, ok := _DISTINCT_AGGREGATES[name]
		return rv, ok
	} else {
		rv, ok := _OTHER_AGGREGATES[name]
		return rv, ok
	}
}

var _DISTINCT_AGGREGATES = map[string]Aggregate{
	"ARRAY_AGG": &ArrayAggDistinct{},
	"AVG":       &AvgDistinct{},
	"COUNT":     &CountDistinct{},
	"SUM":       &SumDistinct{},
}

var _OTHER_AGGREGATES = map[string]Aggregate{
	"ARRAY_AGG": &ArrayAgg{},
	"AVG":       &Avg{},
	"COUNT":     &Count{},
	"MAX":       &Max{},
	"MIN":       &Min{},
	"SUM":       &Sum{},
}
