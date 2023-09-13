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

type IsValued struct {
	UnaryFunctionBase
}

func NewIsValued(operand Expression) Function {
	rv := &IsValued{}
	rv.Init("isvalued", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IsValued) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsValued(this)
}

func (this *IsValued) Type() value.Type { return value.BOOLEAN }

func (this *IsValued) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.NULL, value.MISSING:
		return value.FALSE_VALUE, nil
	default:
		return value.TRUE_VALUE, nil
	}
}

func (this *IsValued) PropagatesMissing() bool {
	return false
}

func (this *IsValued) PropagatesNull() bool {
	return false
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IsValued, simply list this expression.
*/
func (this *IsValued) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *IsValued) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *IsValued) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsValued(operands[0])
	}
}

type IsNotValued struct {
	UnaryFunctionBase
}

func NewIsNotValued(operand Expression) Function {
	rv := &IsNotValued{}
	rv.Init("isnotvalued", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IsNotValued) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNotValued(this)
}

func (this *IsNotValued) Type() value.Type { return value.BOOLEAN }

func (this *IsNotValued) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.NULL, value.MISSING:
		return value.TRUE_VALUE, nil
	default:
		return value.FALSE_VALUE, nil
	}
}

func (this *IsNotValued) PropagatesMissing() bool {
	return false
}

func (this *IsNotValued) PropagatesNull() bool {
	return false
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IsNotValued, simply list this expression.
*/
func (this *IsNotValued) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *IsNotValued) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *IsNotValued) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNotValued(operands[0])
	}
}
