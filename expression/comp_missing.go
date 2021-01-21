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
	"github.com/couchbase/query/value"
)

type IsMissing struct {
	UnaryFunctionBase
}

func NewIsMissing(operand Expression) Function {
	rv := &IsMissing{
		*NewUnaryFunctionBase("ismissing", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IsMissing) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsMissing(this)
}

func (this *IsMissing) Type() value.Type { return value.BOOLEAN }

func (this *IsMissing) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.MISSING:
		return value.TRUE_VALUE, nil
	default:
		return value.FALSE_VALUE, nil
	}
}

func (this *IsMissing) PropagatesMissing() bool {
	return false
}

func (this *IsMissing) PropagatesNull() bool {
	return false
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IsMissing, simply list this expression.
*/
func (this *IsMissing) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.Operand().String()] = value.MISSING_VALUE
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *IsMissing) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsMissing(operands[0])
	}
}

type IsNotMissing struct {
	UnaryFunctionBase
}

func NewIsNotMissing(operand Expression) Function {
	rv := &IsNotMissing{
		*NewUnaryFunctionBase("isnotmissing", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IsNotMissing) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNotMissing(this)
}

func (this *IsNotMissing) Type() value.Type { return value.BOOLEAN }

func (this *IsNotMissing) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.MISSING:
		return value.FALSE_VALUE, nil
	default:
		return value.TRUE_VALUE, nil
	}
}

func (this *IsNotMissing) PropagatesMissing() bool {
	return false
}

func (this *IsNotMissing) PropagatesNull() bool {
	return false
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IsNotMissing, simply list this expression.
*/
func (this *IsNotMissing) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *IsNotMissing) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNotMissing(operands[0])
	}
}
