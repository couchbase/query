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
	"github.com/couchbase/query/plan"
)

type sargLT struct {
	sargBase
}

func newSargLT(pred *expression.LT) *sargLT {
	rv := &sargLT{}
	rv.sarger = func(expr2 expression.Expression) (plan.Spans, error) {
		if SubsetOf(pred, expr2) {
			return _SELF_SPANS, nil
		}

		var exprs expression.Expressions
		span := &plan.Span{}

		if pred.First().EquivalentTo(expr2) {
			exprs = expression.Expressions{pred.Second().Static()}
			span.Range.High = exprs
			span.Range.Low = _NULL_EXPRS
		} else if pred.Second().EquivalentTo(expr2) {
			exprs = expression.Expressions{pred.First().Static()}
			span.Range.Low = exprs
		} else {
			return nil, nil
		}

		if len(exprs) == 0 || exprs[0] == nil {
			return _VALUED_SPANS, nil
		}

		span.Range.Inclusion = datastore.NEITHER
		return plan.Spans{span}, nil
	}

	return rv
}

var _NULL_EXPRS = expression.Expressions{expression.NULL_EXPR}
