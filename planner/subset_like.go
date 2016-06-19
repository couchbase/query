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
	"math"

	"github.com/couchbase/query/expression"
)

type subsetLike struct {
	predicate
}

func newSubsetLike(expr expression.LikeFunction) expression.Visitor {
	re := expr.Regexp()
	if re == nil {
		// Pattern is not a constant
		return newSubsetDefault(expr)
	}

	prefix, complete := re.LiteralPrefix()
	if complete {
		eq := expression.NewEq(expr.First(), expression.NewConstant(prefix))
		return newSubsetEq(eq.(*expression.Eq))
	}

	if prefix == "" {
		return newSubsetDefault(expr)
	}

	var and expression.Expression
	le := expression.NewLE(expression.NewConstant(prefix), expr.First())
	last := len(prefix) - 1
	if prefix[last] < math.MaxUint8 {
		bytes := []byte(prefix)
		bytes[last]++
		and = expression.NewAnd(le, expression.NewLT(
			expr.First(),
			expression.NewConstant(string(bytes))))
	} else {
		and = expression.NewAnd(le, expression.NewLT(
			expr.First(),
			expression.EMPTY_ARRAY_EXPR))
	}

	sand := newSubsetAnd(and.(*expression.And))

	rv := &subsetLike{}
	rv.test = func(expr2 expression.Expression) (bool, error) {
		if expr.EquivalentTo(expr2) {
			return true, nil
		}

		return sand.test(expr2)
	}

	return rv
}
