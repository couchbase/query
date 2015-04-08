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

type sargEq struct {
	sargBase
}

func newSargEq(cond *expression.Eq) *sargEq {
	rv := &sargEq{}
	rv.sarg = func(expr2 expression.Expression) (Spans, error) {
		if SubsetOf(cond, expr2) {
			return _SELF_SPANS, nil
		}

		span := &Span{}

		if cond.First().EquivalentTo(expr2) {
			span.Range.Low = expression.Expressions{cond.Second().Static()}
		} else if cond.Second().EquivalentTo(expr2) {
			span.Range.Low = expression.Expressions{cond.First().Static()}
		} else {
			return nil, nil
		}

		if span.Range.Low[0] == nil {
			return _VALUED_SPANS, nil
		}

		span.Range.Inclusion = datastore.LOW
		hv := span.Range.Low[0].Value()
		if hv != nil {
			hv = hv.Successor()
			if hv != nil {
				span.Range.High = expression.Expressions{expression.NewConstant(hv)}
			}
		}

		return Spans{span}, nil
	}

	return rv
}
