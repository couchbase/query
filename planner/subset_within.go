//  Copyright (c) 2016 Couchbase, Withinc.
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

func (this *subset) VisitWithin(expr *expression.Within) (interface{}, error) {
	switch expr2 := this.expr2.(type) {
	case *expression.IsNotMissing:
		return expr2.Operand().DependsOn(expr.First()), nil
	case *expression.IsNotNull:
		return expr2.Operand().DependsOn(expr.First()), nil
	case *expression.IsValued:
		return expr2.Operand().DependsOn(expr.First()), nil
	default:
		return this.visitDefault(expr)
	}
}
