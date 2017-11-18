//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"strings"
)

/*
This method is used to retrieve an aggregate function by the
parser. Based on the input string name and if DISTINCT is
specified in the query, it looks through a map and retrieves
the aggregate function that corresponds to it. If the function
exists it returns true and the function. While looking into
the map, convert the string name to lowercase.
*/
func GetAggregate(name string, distinct bool) (Aggregate, bool) {
	if distinct {
		rv, ok := _DISTINCT_AGGREGATES[strings.ToLower(name)]
		return rv, ok
	} else {
		rv, ok := _OTHER_AGGREGATES[strings.ToLower(name)]
		return rv, ok
	}
}

/*
Aggregate functions with a DISTINCT specified. The variable
represents a map from string to Aggregate Function. The
aggregate functions ARRAY_AGG, AVG, COUNT and SUM  are
defined by _DISTINCT_AGGREGATES. They map to the corresponding
distinct methods.
*/
var _DISTINCT_AGGREGATES = map[string]Aggregate{
	"array_agg": &ArrayAggDistinct{},
	"avg":       &AvgDistinct{},
	"count":     &CountDistinct{},
	"countn":    &CountnDistinct{},
	"sum":       &SumDistinct{},
}

/*
Non Distinct Aggregate functions. The variable represents a
map from string to Aggregate Function. Contains aggregate
functions ARRAY_AGG, AVG, COUNT, MAX, MIN and SUM.
*/
var _OTHER_AGGREGATES = map[string]Aggregate{
	"array_agg": &ArrayAgg{},
	"avg":       &Avg{},
	"count":     &Count{},
	"countn":    &Countn{},
	"max":       &Max{},
	"min":       &Min{},
	"sum":       &Sum{},
}
