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
	"github.com/couchbaselabs/query/expression"
)

type subsetDefault struct {
	predicate
}

func newSubsetDefault(expr expression.Expression) *subsetDefault {
	rv := &subsetDefault{}
	rv.test = func(expr2 expression.Expression) (bool, error) {
		if expr.EquivalentTo(expr2) {
			return true, nil
		}

		switch expr2 := expr2.(type) {
		case *expression.And:
			for _, op := range expr2.Operands() {
				if !SubsetOf(expr, op) {
					return false, nil
				}
			}

			return true, nil
		case *expression.Or:
			for _, op := range expr2.Operands() {
				if SubsetOf(expr, op) {
					return true, nil
				}
			}

			return false, nil
		case *expression.IsNotMissing:
			return expr.PropagatesMissing() &&
				expr.DependsOn(expr2.Operand()), nil
		case *expression.IsNotNull:
			return expr.PropagatesNull() &&
				expr.DependsOn(expr2.Operand()), nil
		case *expression.IsValued:
			return expr.PropagatesNull() &&
				expr.DependsOn(expr2.Operand()), nil
		}

		return false, nil
	}

	return rv
}
