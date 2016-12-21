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

func (this *subset) visitLike(expr expression.LikeFunction) (interface{}, error) {
	expr2 := this.expr2
	value2 := expr2.Value()
	if value2 != nil {
		return value2.Truth(), nil
	}

	if expr.EquivalentTo(expr2) {
		return true, nil
	}

	re := expr.Regexp()
	if re == nil {
		// Pattern is not a constant
		return this.visitDefault(expr)
	}

	prefix, complete := re.LiteralPrefix()
	if complete {
		eq := expression.NewEq(expr.First(), expression.NewConstant(prefix))
		return eq.Accept(this)
	}

	if prefix == "" {
		return this.visitDefault(expr)
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

	return and.Accept(this)
}
