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

// Unary operators.
type unary interface {
	Expression
	evaluate(operand value.Value) (value.Value, error)
}

type unaryBase struct {
	ExpressionBase
	operand Expression
}

func (this *unaryBase) Evaluate(item value.Value, context Context) (value.Value, error) {
	operand, e := this.operand.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	return unary(this).evaluate(operand)
}

func (this *unaryBase) Fold() (Expression, error) {
	t, e := Expression(this).VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	switch o := this.operand.(type) {
	case *Constant:
		v, e := unary(this).evaluate(o.Value())
		if e == nil {
			return NewConstant(v), nil
		}
	}

	return this, nil
}

func (this *unaryBase) Children() Expressions {
	return Expressions{this.operand}
}

func (this *unaryBase) VisitChildren(visitor Visitor) (Expression, error) {
	var e error
	this.operand, e = visitor.Visit(this.operand)
	if e != nil {
		return nil, e
	}

	return this, nil
}

func (this *unaryBase) evaluate(operand value.Value) (value.Value, error) {
	panic("Must override.")
}
