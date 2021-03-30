//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Abort
//
///////////////////////////////////////////////////

/*
this represents programmatically cancelling a request
*/
type Abort struct {
	UnaryFunctionBase
}

func NewAbort(operand Expression) Function {
	rv := &Abort{
		*NewUnaryFunctionBase("abort", operand),
	}
	rv.setVolatile()

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Abort) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Abort) Type() value.Type {
	return value.JSON
}

func (this *Abort) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return value.NULL_VALUE, errors.NewAbortError(fmt.Sprintf("%v", arg))
}

/*
Factory method pattern.
*/
func (this *Abort) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAbort(operands[0])
	}
}
