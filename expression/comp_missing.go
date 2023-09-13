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

type IsMissing struct {
	UnaryFunctionBase
}

func NewIsMissing(operand Expression) Function {
	rv := &IsMissing{}
	rv.Init("ismissing", operand)

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

func (this *IsMissing) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this.Operand()] = value.MISSING_VALUE
	covers[this] = value.TRUE_VALUE
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
	rv := &IsNotMissing{}
	rv.Init("isnotmissing", operand)

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

func (this *IsNotMissing) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
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
