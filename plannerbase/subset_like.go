//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plannerbase

import (
	"math"
	"regexp"
	"strings"

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

	switch expr2 := expr2.(type) {
	case *expression.Like:
		if _, ok := expr.(*expression.Like); !ok {
			return false, nil
		}
		return likeSubset(expr, expr2, re, prefix)
	case *expression.RegexpLike:
		if _, ok := expr.(*expression.RegexpLike); !ok {
			return false, nil
		}
		return likeSubset(expr, expr2, re, prefix)
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

func likeSubset(expr1, expr2 expression.LikeFunction, re1 *regexp.Regexp, prefix1 string) (interface{}, error) {
	// make sure left-hand side matches
	if expr1 == nil || expr2 == nil || re1 == nil || prefix1 == "" || !expr1.First().EquivalentTo(expr2.First()) {
		return false, nil
	}

	re2 := expr2.Regexp()
	if re2 == nil || re1.NumSubexp() != re2.NumSubexp() {
		return false, nil
	}

	prefix2, complete := re2.LiteralPrefix()
	if complete || prefix2 == "" {
		return false, nil
	}

	// make sure prefix of re1 is superset of (i.e. more restrictive) prefix of re2
	len1 := len(prefix1)
	len2 := len(prefix2)
	if len1 < len2 || prefix1[:len2] != prefix2 {
		return false, nil
	}

	// make sure rest of pattern (excluding the prefix) is identical
	sub1 := strings.Replace(re1.String(), prefix1, "", 1)
	sub2 := strings.Replace(re2.String(), prefix2, "", 1)
	if sub1 != sub2 {
		return false, nil
	}

	return true, nil
}
