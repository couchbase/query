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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
)

type sargLT struct {
	sargBase
}

func newSargLT(expr *expression.LT) *sargLT {
	rv := &sargLT{}
	rv.sarg = func(expr2 expression.Expression) (Spans, error) {
		if expr.EquivalentTo(expr2) {
			return _SELF_SPANS, nil
		}

		var exprs expression.Expressions
		span := &Span{}

		if expr.First().EquivalentTo(expr2) {
			exprs = expression.Expressions{expr.Second().Static()}
			span.Range.High = exprs
		} else if expr.Second().EquivalentTo(expr2) {
			exprs = expression.Expressions{expr.First().Static()}
			span.Range.Low = exprs
		}

		if len(exprs) == 0 {
			return nil, nil
		}

		span.Range.Inclusion = datastore.NEITHER
		return Spans{span}, nil
	}

	return rv
}
