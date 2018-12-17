//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
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
		iset := value.NewSet(len(ivals), false)
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

	default:
		return this.visitDefault(expr)
	}
}
