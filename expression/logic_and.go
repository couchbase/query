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
Standard AND operators are supported.
*/
type And struct {
	CommutativeFunctionBase
}

func NewAnd(operands ...Expression) *And {
	rv := &And{}
	rv.Init("and", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *And) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAnd(this)
}

func (this *And) Type() value.Type { return value.BOOLEAN }

/*
Return FALSE if any known input has a truth value of FALSE, else
return MISSING, NULL, or TRUE in that order.
*/
func (this *And) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false

	for _, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		switch arg.Type() {
		case value.NULL:
			null = true
		case value.MISSING:
			missing = true
		default:
			if !arg.Truth() {
				return value.FALSE_VALUE, nil
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	} else {
		return value.TRUE_VALUE, nil
	}
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For AND, simply cumulate the implicit covers of each child operand.
*/
func (this *And) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	for _, op := range this.operands {
		covers = op.FilterCovers(covers)
	}

	return covers
}

func (this *And) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	for _, op := range this.operands {
		covers = op.FilterExpressionCovers(covers)
	}

	return covers
}

/*
Factory method pattern.
*/
func (this *And) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAnd(operands...)
	}
}

func (this *And) EquivalentTo(other Expression) bool {
	if this.valueEquivalentTo(other) {
		return true
	}
	if a2, ok := other.(*And); ok {
		a1, _ := FlattenAndNoDedup(this)
		a2, _ = FlattenAndNoDedup(a2)
		return a1.CommutativeFunctionBase.doEquivalentTo(a2)
	} else {
		return false
	}
}
