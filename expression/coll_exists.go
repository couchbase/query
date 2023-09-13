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
Represents the Collection expression EXISTS.
*/
type Exists struct {
	UnaryFunctionBase
}

func NewExists(operand Expression) *Exists {
	rv := &Exists{}
	rv.Init("exists", operand)

	rv.expr = rv
	return rv
}

func (this *Exists) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExists(this)
}

func (this *Exists) Type() value.Type { return value.BOOLEAN }

/*
Returns true if the value is an array and contains at least one
element.
*/
func (this *Exists) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.ARRAY {
		a := arg.Actual().([]interface{})
		return value.NewValue(len(a) > 0), nil
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For EXISTS, simply list this expression.
*/
func (this *Exists) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Exists) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

/*
Factory method pattern.
*/
func (this *Exists) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewExists(operands[0])
	}
}
