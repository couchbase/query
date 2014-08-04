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
	"math"

	"github.com/couchbaselabs/query/value"
)

// n-ary operators.
type nAry interface {
	Expression
	eval(operands value.Values) (value.Value, error)
}

type nAryBase struct {
	ExpressionBase
	operands Expressions
}

func (this *nAryBase) evaluate(expr nAry, item value.Value, context Context) (value.Value, error) {
	var e error
	operands := make([]value.Value, len(this.operands))
	for i, o := range this.operands {
		operands[i], e = o.Evaluate(item, context)
		if e != nil {
			return nil, e
		}
	}

	return expr.eval(operands)
}

func (this *nAryBase) fold(expr nAry) (Expression, error) {
	t, e := expr.VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	constants := make(value.Values, len(this.operands))
	for i, o := range this.operands {
		switch o := o.(type) {
		case *Constant:
			constants[i] = o.Value()
		default:
			return expr, nil
		}
	}

	c, e := expr.eval(constants)
	if e != nil {
		return nil, e
	}

	return NewConstant(c), nil
}

func (this *nAryBase) Children() Expressions {
	return this.operands
}

func (this *nAryBase) visitChildren(expr Expression, visitor Visitor) (Expression, error) {
	var err error
	for i, o := range this.operands {
		this.operands[i], err = visitor.Visit(o)
		if err != nil {
			return nil, err
		}
	}

	return expr, nil
}

func (this *nAryBase) MinArgs() int { return 1 }

func (this *nAryBase) MaxArgs() int { return math.MaxInt16 }
