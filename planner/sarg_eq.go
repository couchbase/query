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
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
)

type sargEq struct {
	sargBase
}

func newSargEq(expr *expression.Eq) *sargEq {
	rv := &sargEq{}
	rv.sarg = func(expr2 expression.Expression) (Spans, error) {
		if expr.EquivalentTo(expr2) {
			return _SELF_SPANS, nil
		}

		span := &Span{}

		if expr.First().EquivalentTo(expr2) {
			span.Range.Low = expression.Expressions{expr.Second().Static()}
		} else if expr.Second().EquivalentTo(expr2) {
			span.Range.Low = expression.Expressions{expr.First().Static()}
		}

		if len(span.Range.Low) == 0 {
			return nil, nil
		}

		span.Range.High = span.Range.Low
		span.Range.Inclusion = datastore.BOTH
		return Spans{span}, nil
	}

	return rv
}
