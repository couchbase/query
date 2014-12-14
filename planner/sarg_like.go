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
	"regexp"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
)

type sargLike struct {
	sargBase
}

var _EMPTY_ARRAY = expression.Expressions{expression.EMPTY_ARRAY_EXPR}

func newSargLike(expr expression.BinaryFunction, re *regexp.Regexp) expression.Visitor {
	prefix := ""
	if re != nil {
		prefix, complete := re.LiteralPrefix()
		if complete {
			eq := expression.NewEq(expr.First(), expression.NewConstant(prefix))
			return newSargEq(eq.(*expression.Eq))
		}
	}

	rv := &sargLike{}
	rv.sarg = func(expr2 expression.Expression) (Spans, error) {
		if expr.EquivalentTo(expr2) {
			return _SELF_SPANS, nil
		}

		if !expr.First().EquivalentTo(expr2) {
			return nil, nil
		}

		span := &Span{}
		span.Range.Low = expression.Expressions{expression.NewConstant(prefix)}

		last := len(prefix) - 1
		if last >= 0 && prefix[last] < math.MaxUint8 {
			bytes := []byte(prefix)
			bytes[last]++
			span.Range.High = expression.Expressions{expression.NewConstant(string(bytes))}
		} else {
			span.Range.High = _EMPTY_ARRAY
		}

		span.Range.Inclusion = datastore.LOW
		return Spans{span}, nil
	}

	return rv
}
