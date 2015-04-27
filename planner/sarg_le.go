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

func newSargLE(pred *expression.LE) *sargLE {
	rv := &sargLE{}
	rv.sarger = func(expr2 expression.Expression) (Spans, error) {
		if SubsetOf(pred, expr2) {
			return _SELF_SPANS, nil
		}

		var exprs expression.Expressions
		span := &Span{}

		if pred.First().EquivalentTo(expr2) {
			hs := pred.Second().Static()
			if hs != nil {
				if rv.MissingHigh() {
					exprs = expression.Expressions{expression.NewSuccessor(hs)}
				} else {
					exprs = expression.Expressions{hs}
					span.Range.Inclusion = datastore.HIGH
				}

				span.Range.High = exprs
			}
		} else if pred.Second().EquivalentTo(expr2) {
			exprs = expression.Expressions{pred.First().Static()}
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
