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

type sargLE struct {
	sargBase
}

func newSargLE(cond *expression.LE) *sargLE {
	rv := &sargLE{}
	rv.sarg = func(expr2 expression.Expression) (Spans, error) {
		if SubsetOf(cond, expr2) {
			return _SELF_SPANS, nil
		}

		var exprs expression.Expressions
		span := &Span{}

		if cond.First().EquivalentTo(expr2) {
			hs := cond.Second().Static()
			if hs != nil {
				hv := hs.Value()
				if hv != nil {
					hv = hv.Successor()
					if hv != nil {
						exprs = expression.Expressions{expression.NewConstant(hv)}
						span.Range.High = exprs
						span.Range.Inclusion = datastore.HIGH
					}
				}
			}
		} else if cond.Second().EquivalentTo(expr2) {
			exprs = expression.Expressions{cond.First().Static()}
			span.Range.Low = exprs
			span.Range.Inclusion = datastore.LOW
		} else {
			return nil, nil
		}

		if len(exprs) == 0 || exprs[0] == nil {
			return _VALUED_SPANS, nil
		}

		return Spans{span}, nil
	}

	return rv
}
