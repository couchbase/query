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
)

type sargOr struct {
	sargDefault
}

func newSargOr(cond *expression.Or) *sargOr {
	rv := &sargOr{}
	rv.sarg = func(expr2 expression.Expression) (Spans, error) {
		if cond.EquivalentTo(expr2) {
			return _SELF_SPANS, nil
		}

		spans := make(Spans, 0, len(cond.Operands()))
		for _, child := range cond.Operands() {
			cspans := SargFor(child, expression.Expressions{expr2})
			if len(cspans) > 0 {
				spans = append(spans, cspans...)
			}
		}

		return spans, nil
	}

	return rv
}
