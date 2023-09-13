//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"math"

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Greatest
//
///////////////////////////////////////////////////

/*
This represents the comparison function GREATEST(expr1, expr2, ...).
It returns the largest non-NULL, non-MISSING input value.
*/
type Greatest struct {
	FunctionBase
}

func NewGreatest(operands ...Expression) Function {
	rv := &Greatest{}
	rv.Init("greatest", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Greatest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Greatest) Type() value.Type { return value.JSON }

func (this *Greatest) Evaluate(item value.Value, context Context) (value.Value, error) {
	rv := value.NULL_VALUE
	for _, op := range this.operands {
		a, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if a.Type() <= value.NULL {
			continue
		} else if rv == value.NULL_VALUE {
			rv = a
		} else if a.Collate(rv) > 0 {
			rv = a
		}
	}

	return rv, nil
}

/*
Minimum input arguments required for the defined function
GREATEST is 2.
*/
func (this *Greatest) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the GREATEST
function is MaxInt16  = 1<<15 - 1.
*/
func (this *Greatest) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *Greatest) Constructor() FunctionConstructor {
	return NewGreatest
}

///////////////////////////////////////////////////
//
// Least
//
///////////////////////////////////////////////////

/*
This represents the comparison function LEAST(expr1, expr2, ...). It
returns the smallest non-NULL, non-MISSING input value.
*/
type Least struct {
	FunctionBase
}

func NewLeast(operands ...Expression) Function {
	rv := &Least{}
	rv.Init("least", operands...)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Least) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Least) Type() value.Type { return value.JSON }

func (this *Least) Evaluate(item value.Value, context Context) (value.Value, error) {
	rv := value.NULL_VALUE

	for _, op := range this.operands {
		a, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if a.Type() <= value.NULL {
			continue
		} else if rv == value.NULL_VALUE {
			rv = a
		} else if a.Collate(rv) < 0 {
			rv = a
		}
	}

	return rv, nil
}

/*
Minimum input arguments required for the defined function
LEAST is 2.
*/
func (this *Least) MinArgs() int { return 2 }

/*
Maximum number of input arguments defined for the LEAST
function is MaxInt16  = 1<<15 - 1.
*/
func (this *Least) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *Least) Constructor() FunctionConstructor {
	return NewLeast
}

///////////////////////////////////////////////////
//
// Successor
//
///////////////////////////////////////////////////

/*
This Expression is primarily for internal use. It returns a successor
to the input argument, in N1QL collation order.
*/
type Successor struct {
	UnaryFunctionBase
}

func NewSuccessor(operand Expression) Function {
	rv := &Successor{}
	rv.Init("successor", operand)

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Successor) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Successor) Type() value.Type {
	return this.Operand().Type().Successor()
}

func (this *Successor) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	return arg.Successor(), nil
}

/*
Factory method pattern.
*/
func (this *Successor) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSuccessor(operands[0])
	}
}
