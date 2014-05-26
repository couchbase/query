//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbaselabs/query/value"
)

type Exists struct {
	ExpressionBase
	operand Expression
}

func NewExists(operand Expression) *Exists {
	return &Exists{
		operand: operand,
	}
}

func (this *Exists) Evaluate(item value.Value, context Context) (value.Value, error) {
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

func (this *Exists) Fold() (Expression, error) {
	t, e := Expression(this).VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	switch o := this.operand.(type) {
	case *Constant:
		v, e := this.Evaluate(o.Value(), nil)
		if e == nil {
			return NewConstant(v), nil
		}
	}

	return this, nil
}

func (this *Exists) Children() Expressions {
	return Expressions{this.operand}
}

func (this *Exists) VisitChildren(visitor Visitor) (Expression, error) {
	var e error
	this.operand, e = visitor.Visit(this.operand)
	if e != nil {
		return nil, e
	}

	return this, nil
}

func (this *Exists) Operand() Expression {
	return this.operand
}

var _ONE = value.NewValue(1)
var _ONE_EXPR = NewConstant(_ONE)
