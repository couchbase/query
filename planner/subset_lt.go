//  Copyright (c) 2014 Couchbase, Inc.
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
)

func (this *subset) VisitLT(expr *expression.LT) (interface{}, error) {
	switch expr2 := this.expr2.(type) {

	case *expression.LE:
		if expr.First().EquivalentTo(expr2.First()) {
			return LessThanOrEquals(expr.Second(), expr2.Second()), nil
		}

		if expr.Second().EquivalentTo(expr2.Second()) {
			return LessThanOrEquals(expr2.First(), expr.First()), nil
		}

		return false, nil

	case *expression.LT:
		if expr.First().EquivalentTo(expr2.First()) {
			return LessThanOrEquals(expr.Second(), expr2.Second()), nil
		}

		if expr.Second().EquivalentTo(expr2.Second()) {
			return LessThanOrEquals(expr2.First(), expr.First()), nil
		}

		return false, nil

	default:
		return this.visitDefault(expr)
	}
}
