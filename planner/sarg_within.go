//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func (this *sarg) VisitWithin(pred *expression.Within) (interface{}, error) {
	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	if !pred.First().EquivalentTo(this.key) {
		if pred.DependsOn(this.key) {
			return getDependsSpans(pred)
		} else {
			return nil, nil
		}
	}

	aval := pred.Second().Value()
	if aval == nil {
		return _VALUED_SPANS, nil
	}

	array, ok := aval.Actual().([]interface{})
	if ok {
		array = _WITHIN_POOL.GetCapped(len(array))
	} else {
		array = _WITHIN_POOL.GetCapped(_WITHIN_POOL_SIZE)
	}
	defer _WITHIN_POOL.Put(array)

	array = aval.Descendants(array)
	if len(array) == 0 {
		return _EMPTY_SPANS, nil
	}

	// De-dup before generating spans
	set := value.NewSet(len(array), true, false)
	set.AddAll(array)
	array = set.Actuals()

	// Sort for EXPLAIN stability
	sort.Sort(value.NewSorter(value.NewValue(array)))

	spans := make(plan.Spans2, 0, len(array))
	for _, val := range array {
		if val == nil {
			continue
		}

		selec := OPT_SELEC_NOT_AVAIL
		if this.doSelec {
			selec = optDefInSelec(this.baseKeyspace.Keyspace(), this.key.String(), this.advisorValidate)
		}
		expr := expression.NewConstant(val)
		range2 := plan.NewRange2(expr, expr, datastore.BOTH, selec, OPT_SELEC_NOT_AVAIL, 0)
		span := plan.NewSpan2(nil, plan.Ranges2{range2}, true)
		spans = append(spans, span)
	}

	return NewTermSpans(spans...), nil
}

const _WITHIN_POOL_SIZE = 64

var _WITHIN_POOL = util.NewInterfacePool(_WITHIN_POOL_SIZE)
