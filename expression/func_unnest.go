//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// UnnestPosition
//
///////////////////////////////////////////////////

/*
UNNEST_POSITION(expr)
*/
type UnnestPosition struct {
	UnaryFunctionBase
}

func NewUnnestPosition(operand Expression) Function {
	rv := &UnnestPosition{}
	rv.Init("unnest_position", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *UnnestPosition) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *UnnestPosition) Type() value.Type { return value.NUMBER }

func (this *UnnestPosition) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	av, ok := arg.(value.AnnotatedValue)
	if !ok {
		return value.NULL_VALUE, nil
	}

	upos := av.GetAttachment("unnest_position")
	if upos == nil {
		return value.NULL_VALUE, nil
	}

	pos, ok := upos.(int)
	if !ok {
		return value.NULL_VALUE, errors.NewUnnestInvalidPosition(pos)
	}

	return value.NewValue(pos), nil
}

/*
Factory method pattern.
*/
func (this *UnnestPosition) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewUnnestPosition(operands[0])
	}
}
