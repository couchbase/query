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
	"github.com/couchbaselabs/query/value"
)

type Exists struct {
	expression.ExpressionBase
	operand expression.Expression
}

func NewExists(operand expression.Expression) expression.Expression {
	return &Exists{
		operand: operand,
	}
}

func (this *Exists) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	operand, e := this.operand.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if operand.Type() == value.ARRAY {
		a := operand.Actual().([]interface{})
		return value.NewValue(len(a) > 0), nil
	} else if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

func (this *Exists) Dependencies() expression.Expressions {
	return expression.Expressions{this.operand}
}

func (this *Exists) Fold() expression.Expression {
	this.operand = this.operand.Fold()
	switch o := this.operand.(type) {
	case *expression.Constant:
		v, e := this.Evaluate(o.Value(), nil)
		if e == nil {
			return expression.NewConstant(v)
		}
	case *Subquery:
		o.query.SetLimit(_ONE_EXPR)
	}

	return this
}

var _ONE_EXPR = expression.NewConstant(_ONE)
