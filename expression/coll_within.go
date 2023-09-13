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
Represents the collection expression WITHIN.
*/
type Within struct {
	BinaryFunctionBase
}

func NewWithin(first, second Expression) Function {
	rv := &Within{}
	rv.Init("within", first, second)

	rv.expr = rv
	return rv
}

func (this *Within) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWithin(this)
}

func (this *Within) Type() value.Type { return value.BOOLEAN }

/*
WITHIN evaluates to TRUE if the right-hand-side first value contains
the left-hand-side second value (or name and value) as a child or
descendant (i.e. directly or indirectly).
*/
func (this *Within) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.ARRAY && second.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	var missing, null bool
	buf := _INTERFACE_POOL.Get()
	defer _INTERFACE_POOL.Put(buf)
	desc := second.Descendants(buf)
	for _, d := range desc {
		v := value.NewValue(d)
		if first.Type() > value.NULL && v.Type() > value.NULL {
			if first.Equals(v).Truth() {
				return value.TRUE_VALUE, nil
			}
		} else if v.Type() == value.MISSING {
			missing = true
		} else {
			// first.Type() == value.NULL || v.Type() == value.NULL
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	} else if missing {
		return value.MISSING_VALUE, nil
	} else {
		return value.FALSE_VALUE, nil
	}
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For WITHIN, simply list this expression.
*/
func (this *Within) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Within) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

func (this *Within) MayOverlapSpans() bool {
	return this.Second().Value() == nil
}

/*
Factory method pattern.
*/
func (this *Within) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewWithin(operands[0], operands[1])
	}
}

/*
This function implements the NOT WITHIN collection operation.
*/
func NewNotWithin(first, second Expression) Expression {
	return NewNot(NewWithin(first, second))
}
