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
Comparison terms allow for comparing two expressions. For
EQUALS (= and ==) and NOT EQUALS (!= and <>) two forms
are supported to aid in compatibility with other query
languages.
*/
type Eq struct {
	CommutativeBinaryFunctionBase
}

func NewEq(first, second Expression) Function {
	rv := &Eq{}
	rv.Init("eq", first, second)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Eq) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitEq(this)
}

/*
It returns a value type BOOLEAN.
*/
func (this *Eq) Type() value.Type { return value.BOOLEAN }

func (this *Eq) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	return first.Equals(second), nil
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For Eq, list either a static value, or this expression.
*/
func (this *Eq) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	var static, other Expression
	if this.Second().Value() != nil {
		static = this.Second()
		other = this.First()
	} else if this.First().Value() != nil {
		static = this.First()
		other = this.Second()
	}

	if static != nil {
		covers[other.String()] = static.Value()
		return covers
	}

	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *Eq) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	var static, other Expression
	if this.Second().Value() != nil {
		static = this.Second()
		other = this.First()
	} else if this.First().Value() != nil {
		static = this.First()
		other = this.Second()
	}

	if static != nil {
		covers[other] = static.Value()
	} else {
		covers[this] = value.TRUE_VALUE
	}

	return covers
}

/*
Factory method pattern.
*/
func (this *Eq) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewEq(operands[0], operands[1])
	}
}

/*
This function implements the NOT EQUALS comparison operation.
*/
func NewNE(first, second Expression) Expression {
	return NewNot(NewEq(first, second))
}
