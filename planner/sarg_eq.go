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

type sargEq struct {
	sargBase
}

func newSargEq(pred *expression.Eq) *sargEq {
	rv := &sargEq{}
	rv.sarger = func(expr2 expression.Expression) (plan.Spans, error) {
		if SubsetOf(pred, expr2) {
			return _SELF_SPANS, nil
		}

		span := &plan.Span{}

		if pred.First().EquivalentTo(expr2) {
			span.Range.Low = expression.Expressions{pred.Second().Static()}
		} else if pred.Second().EquivalentTo(expr2) {
			span.Range.Low = expression.Expressions{pred.First().Static()}
		} else {
			return nil, nil
		}

		if span.Range.Low[0] == nil {
			return _VALUED_SPANS, nil
		}

		if rv.MissingHigh() {
			span.Range.High = expression.Expressions{expression.NewSuccessor(span.Range.Low[0])}
			span.Range.Inclusion = datastore.LOW
		} else {
			span.Range.High = span.Range.Low
			span.Range.Inclusion = datastore.BOTH
		}

		return plan.Spans{span}, nil
	}

	return rv
}
