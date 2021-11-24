//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/value"
)

func (this *subset) VisitIn(expr *expression.In) (interface{}, error) {
	switch expr2 := this.expr2.(type) {
	case *expression.Within:
		return expr.First().EquivalentTo(expr2.First()) &&
			expr.Second().EquivalentTo(expr2.Second()), nil
	case *expression.In:
		// Check left side of IN is same
		if !expr.First().EquivalentTo(expr2.First()) {
			return false, nil
		}

		// Check right side of the IN is same, if same we are done
		if expr.Second().EquivalentTo(expr2.Second()) {
			return true, nil
		}

		qval := expr.Second().Value()
		ival := expr2.Second().Value()

		// right side of IN in the index and query must be constants
		if qval == nil || ival == nil {
			return false, nil
		}

		qvals, qok := qval.Actual().([]interface{})
		ivals, iok := ival.Actual().([]interface{})

		// right side of IN in the index and query must be arrays of length > 0
		if !qok || !iok || len(qvals) == 0 || len(ivals) == 0 {
			return false, nil
		}

		// Build values of index
		iset := value.NewSet(len(ivals), false, false)
		for _, v := range ivals {
			iv := value.NewValue(v)
			iset.Put(iv, iv)
		}

		// Check every query value is present in the index
		for _, v := range qvals {
			if !iset.Has(value.NewValue(v)) {
				// query array element is not present in index array
				return false, nil
			}
		}

		// all query array elements are present in index array
		return true, nil

	case *search.Search:
		if expr.First().EquivalentTo(expr2) {
			val := expr.Second().Value()
			if val != nil {
				if vals, ok := val.Actual().([]interface{}); ok && len(vals) > 0 {
					for _, v := range vals {
						iv := value.NewValue(v)
						if iv.Type() != value.BOOLEAN || !iv.Truth() {
							return false, nil
						}
					}
					return true, nil
				}
			}
		}

		return this.visitDefault(expr)

	default:
		return this.visitDefault(expr)
	}
}
