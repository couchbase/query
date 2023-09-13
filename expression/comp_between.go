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

/*
Comparison terms allow for comparing two expressions.
For BETWEEN and NOT BETWEEN, we have three expressions,
the input item and the low and high expressions.
*/
type Between struct {
	TernaryFunctionBase
}

func NewBetween(item, low, high Expression) Function {
	rv := &Between{}
	rv.Init("between", item, low, high)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Between) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBetween(this)
}

func (this *Between) Type() value.Type { return value.BOOLEAN }

func (this *Between) Evaluate(item value.Value, context Context) (value.Value, error) {
	op, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	low, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	high, err := this.operands[2].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	lowCmp := op.Compare(low)
	if lowCmp.Type() == value.MISSING {
		return lowCmp, nil
	}

	highCmp := op.Compare(high)
	if highCmp.Type() == value.MISSING {
		return highCmp, nil
	}

	switch lowActual := lowCmp.Actual().(type) {
	case float64:
		switch highActual := highCmp.Actual().(type) {
		case float64:
			return value.NewValue(lowActual >= 0 && highActual <= 0), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For Between, simply list this expression.
*/
func (this *Between) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Between) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *Between) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBetween(operands[0], operands[1], operands[2])
	}
}

/*
This function implements the NOT BETWEEN operation.
*/
func NewNotBetween(item, low, high Expression) Expression {
	return NewNot(NewBetween(item, low, high))
}
