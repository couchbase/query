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

type Negate struct {
	unaryBase
}

func NewNegate(operand Expression) Expression {
	return &Negate{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *Negate) Fold() (Expression, error) {
	t, e := Expression(this).VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	switch o := this.operand.(type) {
	case *Constant:
		v, e := this.evaluate(o.Value())
		if e != nil {
			return nil, e
		}
		return NewConstant(v), nil
	case *Negate:
		return o.operand, nil
	case *Add:
		operands := make(Expressions, len(o.operands))
		for i, oo := range o.operands {
			operands[i] = NewNegate(oo)
		}
		add := NewAdd(operands...)
		return add.Fold()
	}

	return this, nil
}

func (this *Negate) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.NUMBER {
		return value.NewValue(-operand.Actual().(float64)), nil
	} else if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}
