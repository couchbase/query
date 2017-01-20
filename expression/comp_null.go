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

type IsNull struct {
	UnaryFunctionBase
}

func NewIsNull(operand Expression) Function {
	rv := &IsNull{
		*NewUnaryFunctionBase("isnull", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IsNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNull(this)
}

func (this *IsNull) Type() value.Type { return value.BOOLEAN }

func (this *IsNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsNull) PropagatesNull() bool {
	return false
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IsNull, simply list this expression.
*/
func (this *IsNull) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.Operand().String()] = value.NULL_VALUE
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *IsNull) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.NULL:
		return value.TRUE_VALUE, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.FALSE_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *IsNull) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNull(operands[0])
	}
}

type IsNotNull struct {
	UnaryFunctionBase
}

func NewIsNotNull(operand Expression) Function {
	rv := &IsNotNull{
		*NewUnaryFunctionBase("isnotnull", operand),
	}

	rv.expr = rv
	return rv
}

func (this *IsNotNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNotNull(this)
}

func (this *IsNotNull) Type() value.Type { return value.BOOLEAN }

func (this *IsNotNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsNotNull) PropagatesNull() bool {
	return false
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IsNotNull, simply list this expression.
*/
func (this *IsNotNull) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *IsNotNull) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.NULL:
		return value.FALSE_VALUE, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.TRUE_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *IsNotNull) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNotNull(operands[0])
	}
}
