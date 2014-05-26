//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbaselabs/query/expression"
)

// Exists inherits from expression.Exists to set LIMIT 1 on
// subqueries.
type Exists struct {
	expression.Exists
}

func NewExists(operand expression.Expression) *Exists {
	return &Exists{
		*(expression.NewExists(operand)),
	}
}

// Fold() overrides expression.Exists.Fold() to set LIMIT 1 on
// subqueries.
func (this *Exists) Fold() (expression.Expression, error) {
	t, e := expression.Expression(this).VisitChildren(&expression.Folder{})
	if e != nil {
		return t, e
	}

	switch o := this.Operand().(type) {
	case *expression.Constant:
		v, e := this.Evaluate(o.Value(), nil)
		if e == nil {
			return expression.NewConstant(v), nil
		}
	case *Subquery:
		o.query.SetLimit(_ONE_EXPR)
	}

	return this, nil
}

var _ONE_EXPR = expression.NewConstant(_ONE)
