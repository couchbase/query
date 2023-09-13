//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/value"
)

type IsNull struct {
	UnaryFunctionBase
}

func NewIsNull(operand Expression) Function {
	rv := &IsNull{}
	rv.Init("isnull", operand)

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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.NULL:
		return value.TRUE_VALUE, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.FALSE_VALUE, nil
	}
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

func (this *IsNull) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this.Operand()] = value.NULL_VALUE
	covers[this] = value.TRUE_VALUE
	return covers
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
	rv := &IsNotNull{}
	rv.Init("isnotnull", operand)

	rv.expr = rv
	return rv
}

func (this *IsNotNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNotNull(this)
}

func (this *IsNotNull) Type() value.Type { return value.BOOLEAN }

func (this *IsNotNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.NULL:
		return value.FALSE_VALUE, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.TRUE_VALUE, nil
	}
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

func (this *IsNotNull) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *IsNotNull) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNotNull(operands[0])
	}
}
