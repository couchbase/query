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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

type sargOr struct {
	sargDefault
}

func newSargOr(pred *expression.Or) *sargOr {
	rv := &sargOr{}
	rv.sarger = func(expr2 expression.Expression) (plan.Spans, error) {
		if SubsetOf(pred, expr2) {
			return _SELF_SPANS, nil
		}

		spans := make(plan.Spans, 0, len(pred.Operands()))
		for _, child := range pred.Operands() {
			cspans, err := sargFor(child, expr2, rv.MissingHigh())
			if err != nil {
				return nil, err
			}

			if len(cspans) == 0 {
				return nil, nil
			}

			if cspans[0] == _EXACT_FULL_SPANS[0] {
				return _EXACT_FULL_SPANS, nil
			}

			if cspans[0] == _FULL_SPANS[0] || len(spans)+len(cspans) > _FULL_SPAN_FANOUT {
				return _FULL_SPANS, nil
			}

			spans = append(spans, cspans...)
		}

		return spans, nil
	}

	return rv
}
