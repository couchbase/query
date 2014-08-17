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

type Reciprocate struct {
	unaryBase
}

func NewReciprocate(operand Expression) Expression {
	return &Reciprocate{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *Reciprocate) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Reciprocate) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Reciprocate) Fold() (Expression, error) {
	t, e := this.VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	switch o := this.operand.(type) {
	case *Constant:
		v, e := this.eval(o.Value())
		if e != nil {
			return nil, e
		}
		return NewConstant(v), nil
	case *Reciprocate:
		return o.operand, nil
	case *Multiply:
		operands := make(Expressions, len(o.operands))
		for i, oo := range o.operands {
			operands[i] = NewReciprocate(oo)
		}
		mult := NewMultiply(operands...)
		return mult.Fold()
	}

	return this, nil
}

func (this *Reciprocate) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Reciprocate) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Reciprocate) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Reciprocate) eval(operand value.Value) (value.Value, error) {
	if operand.Type() == value.NUMBER {
		a := operand.Actual().(float64)
		if a == 0.0 {
			return value.NULL_VALUE, nil
		}
		return value.NewValue(1.0 / a), nil
	} else if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}
