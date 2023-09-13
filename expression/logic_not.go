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
Logical terms allow for combining other expressions using boolean logic.
Standard NOT operators are supported.
*/
type Not struct {
	UnaryFunctionBase
}

func NewNot(operand Expression) Function {
	rv := &Not{}
	rv.Init("not", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Not) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNot(this)
}

func (this *Not) Type() value.Type { return value.BOOLEAN }

func (this *Not) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	switch arg.Type() {
	case value.MISSING, value.NULL:
		return arg, nil
	default:
		if arg.Truth() {
			return value.FALSE_VALUE, nil
		} else {
			return value.TRUE_VALUE, nil
		}
	}
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For NOT, simply list this expression.
*/
func (this *Not) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Not) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *Not) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNot(operands[0])
	}
}
