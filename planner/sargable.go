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

func SargableFor(cond expression.Expression, exprs expression.Expressions) bool {
	if len(exprs) == 0 || exprs[0].Value() != nil {
		return false
	}

	s := newSargable(cond)
	result, _ := exprs[0].Accept(s)
	return result.(bool)
}

func newSargable(cond expression.Expression) expression.Visitor {
	s, _ := cond.Accept(_SARGABLE_FACTORY)
	return s.(expression.Visitor)
}
