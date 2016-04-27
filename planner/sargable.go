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

func SargableFor(pred expression.Expression, exprs expression.Expressions) int {
	s := newSargable(pred)
	n := len(exprs)

	if n > 1 {
		// Only AND / OR predicates can sarg more than one index key
		switch pred.(type) {
		case *expression.And, *expression.Or:
		default:
			n = 1
		}
	}

	i := 0
	for ; i < n; i++ {
		// Terminate on statically-valued expression
		if exprs[i].Value() != nil {
			return i
		}

		r, err := exprs[i].Accept(s)
		if err != nil || !r.(bool) {
			return i
		}
	}

	return i
}

func newSargable(pred expression.Expression) expression.Visitor {
	s, _ := pred.Accept(_SARGABLE_FACTORY)
	return s.(expression.Visitor)
}
