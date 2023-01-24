//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"strings"
)

/*
This method is used to retrieve an aggregate function by the
parser. Based on the input string name, aggregate modifier
has DISTINCT  and window aggregate. It looks through a map
and retrieves the aggregate function that corresponds to it.
If the function exists it returns true and the function.
While looking into the map, convert the string name to lowercase.
*/

func GetAggregate(name string, distinct, filter, window bool) (Aggregate, bool) {
	rv, ok := _AGGREGATES[strings.ToLower(name)]
	if !ok || (distinct && !rv.HasProperty(AGGREGATE_ALLOWS_DISTINCT)) ||
		(filter && !rv.HasProperty(AGGREGATE_ALLOWS_FILTER)) ||
		(window && !rv.HasProperty(AGGREGATE_ALLOWS_WINDOW)) {
		return nil, false
	} else if window {
		return rv.agg, rv.HasProperty(AGGREGATE_ALLOWS_WINDOW)
	} else {
		return rv.agg, rv.HasProperty(AGGREGATE_ALLOWS_REGULAR)
	}
}

/*
Aggregate modifers/flags
*/
const (
	AGGREGATE_DISTINCT = 1 << iota
	AGGREGATE_INCREMENTAL
	AGGREGATE_RESPECTNULLS
	AGGREGATE_IGNORENULLS
	AGGREGATE_FROMFIRST
	AGGREGATE_FROMLAST
)

/*
Aggregate properties. These allow syntax and semantics checks.
*/

const (
	AGGREGATE_ALLOWS_REGULAR = 1 << iota
	AGGREGATE_ALLOWS_DISTINCT
	AGGREGATE_ALLOWS_FILTER
	AGGREGATE_ALLOWS_WINDOW
	AGGREGATE_ALLOWS_WINDOW_FRAME
	AGGREGATE_ALLOWS_INCREMENTAL
	AGGREGATE_WINDOW_NOORDER
	AGGREGATE_WINDOW_ORDER
	AGGREGATE_WINDOW_RELEASE_CURRENTROW
	AGGREGATE_WINDOW_RESPECTNULLS
	AGGREGATE_WINDOW_IGNORENULLS
	AGGREGATE_WINDOW_FROMFIRST
	AGGREGATE_WINDOW_FROMLAST
	AGGREGATE_WINDOW_2ND_POSINT
	AGGREGATE_WINDOW_2ND_DYNAMIC
)

/*
Grouped Aggregate properties.
*/
const (
	AGGREGATE_ALLOWS_ALL             = AGGREGATE_ALLOWS_REGULAR | AGGREGATE_ALLOWS_DISTINCT | AGGREGATE_ALLOWS_WINDOW | AGGREGATE_ALLOWS_WINDOW_FRAME | AGGREGATE_ALLOWS_FILTER
	AGGREGATE_ALLOWS_ALL_INCREMENTAL = AGGREGATE_ALLOWS_ALL | AGGREGATE_ALLOWS_INCREMENTAL
	AGGREGATE_WINDOW_RANK            = AGGREGATE_ALLOWS_WINDOW | AGGREGATE_ALLOWS_INCREMENTAL | AGGREGATE_WINDOW_ORDER
	AGGREGATE_ROW_NUMBER             = AGGREGATE_ALLOWS_WINDOW | AGGREGATE_ALLOWS_INCREMENTAL | AGGREGATE_WINDOW_RELEASE_CURRENTROW
	AGGREGATE_ALLOWS_FL              = AGGREGATE_ALLOWS_WINDOW | AGGREGATE_ALLOWS_WINDOW_FRAME | AGGREGATE_WINDOW_RESPECTNULLS | AGGREGATE_WINDOW_IGNORENULLS
	AGGREGATE_ALLOWS_NTH             = AGGREGATE_ALLOWS_FL | AGGREGATE_WINDOW_FROMFIRST | AGGREGATE_WINDOW_FROMLAST | AGGREGATE_WINDOW_2ND_POSINT | AGGREGATE_WINDOW_2ND_DYNAMIC
	AGGREGATE_ALLOWS_LAGLEAD         = AGGREGATE_ALLOWS_WINDOW | AGGREGATE_WINDOW_ORDER | AGGREGATE_WINDOW_RESPECTNULLS | AGGREGATE_WINDOW_IGNORENULLS | AGGREGATE_WINDOW_2ND_POSINT
)

/*
Attachment name that conatins window info
*/
const WINDOW_ATTACHMENT = "window_attachment"

/*
Aggregate registry
*/
type AggregateRegistry struct {
	property uint32
	agg      Aggregate
}

/*
 Returns true if aggregate has property
*/

func (this *AggregateRegistry) HasProperty(p uint32) bool {
	return (this.property & p) != 0
}

/*
 Returns true if given aggregate name has property
*/

func AggregateHasProperty(name string, p uint32) bool {
	rv, ok := _AGGREGATES[strings.ToLower(name)]
	return ok && rv.HasProperty(p)
}

/*
Aggregate functions. The variable represents a map from string to Aggregate Function.
*/

var _AGGREGATES = map[string]*AggregateRegistry{
	"array_agg":       &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &ArrayAgg{}},
	"avg":             &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL_INCREMENTAL, agg: &Avg{}},
	"count":           &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL_INCREMENTAL, agg: &Count{}},
	"countn":          &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL_INCREMENTAL, agg: &Countn{}},
	"max":             &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &Max{}},
	"mean":            &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL_INCREMENTAL, agg: &Avg{}},
	"median":          &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &Median{}},
	"min":             &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &Min{}},
	"stddev":          &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &Stddev{}},
	"stddev_pop":      &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &StddevPop{}},
	"stddev_samp":     &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &StddevSamp{}},
	"sum":             &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL_INCREMENTAL, agg: &Sum{}},
	"variance":        &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &Variance{}},
	"var_pop":         &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &VarPop{}},
	"variance_pop":    &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &VarPop{}},
	"var_samp":        &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &VarSamp{}},
	"variance_samp":   &AggregateRegistry{property: AGGREGATE_ALLOWS_ALL, agg: &VarSamp{}},
	"row_number":      &AggregateRegistry{property: AGGREGATE_ROW_NUMBER, agg: &RowNumber{}},
	"rank":            &AggregateRegistry{property: AGGREGATE_WINDOW_RANK | AGGREGATE_WINDOW_RELEASE_CURRENTROW, agg: &Rank{}},
	"dense_rank":      &AggregateRegistry{property: AGGREGATE_WINDOW_RANK | AGGREGATE_WINDOW_RELEASE_CURRENTROW, agg: &DenseRank{}},
	"percent_rank":    &AggregateRegistry{property: AGGREGATE_WINDOW_RANK, agg: &PercentRank{}},
	"cume_dist":       &AggregateRegistry{property: AGGREGATE_WINDOW_RANK, agg: &CumeDist{}},
	"ratio_to_report": &AggregateRegistry{property: AGGREGATE_ALLOWS_WINDOW | AGGREGATE_ALLOWS_WINDOW_FRAME, agg: &RatioToReport{}},
	"ntile":           &AggregateRegistry{property: AGGREGATE_ALLOWS_WINDOW | AGGREGATE_WINDOW_ORDER, agg: &Ntile{}},
	"first_value":     &AggregateRegistry{property: AGGREGATE_ALLOWS_FL, agg: &FirstValue{}},
	"last_value":      &AggregateRegistry{property: AGGREGATE_ALLOWS_FL, agg: &LastValue{}},
	"nth_value":       &AggregateRegistry{property: AGGREGATE_ALLOWS_NTH, agg: &NthValue{}},
	"lag":             &AggregateRegistry{property: AGGREGATE_ALLOWS_LAGLEAD, agg: &Lag{}},
	"lead":            &AggregateRegistry{property: AGGREGATE_ALLOWS_LAGLEAD, agg: &Lead{}},
}
