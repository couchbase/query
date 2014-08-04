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
	eval(operand value.Value) (value.Value, error)
}

type unaryBase struct {
	ExpressionBase
	operand Expression
}

func (this *unaryBase) evaluate(expr unary, item value.Value, context Context) (value.Value, error) {
	operand, e := this.operand.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	return expr.eval(operand)
}

func (this *unaryBase) fold(expr unary) (Expression, error) {
	t, e := expr.VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	switch o := this.operand.(type) {
	case *Constant:
		v, e := expr.eval(o.Value())
		if e == nil {
			return NewConstant(v), nil
		}
	}

	return expr, nil
}

func (this *unaryBase) Children() Expressions {
	return Expressions{this.operand}
}

func (this *unaryBase) visitChildren(expr Expression, visitor Visitor) (Expression, error) {
	var err error
	this.operand, err = visitor.Visit(this.operand)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *unaryBase) MinArgs() int { return 1 }

func (this *unaryBase) MaxArgs() int { return 1 }
