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
This represents the LESS THAN comparison
operation.
*/
type LT struct {
	BinaryFunctionBase
}

func NewLT(first, second Expression) Function {
	rv := &LT{}
	rv.Init("lt", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *LT) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLT(this)
}

func (this *LT) Type() value.Type { return value.BOOLEAN }

func (this *LT) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	cmp := first.Compare(second)
	switch actual := cmp.Actual().(type) {
	case float64:
		return value.NewValue(actual < 0), nil
	}

	return cmp, nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For LT, simply list this expression.
*/
func (this *LT) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *LT) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *LT) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLT(operands[0], operands[1])
	}
}
