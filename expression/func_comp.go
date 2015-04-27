//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
This represents the comparison function GREATEST(expr1, expr2..).
It is the largest non-NULL, non-MISSING value if the values are
of the same type, otherwise NULL. Type Greatest is a struct that
implements FunctionBase.
*/
type Greatest struct {
	FunctionBase
}

/*
The function NewGreatest takes as input expressions and returns
a pointer to the Greatest struct that calls NewFunctionBase to
create a function named GREATEST with input operands as the
expressions.
*/
func NewGreatest(operands ...Expression) Function {
	rv := &Greatest{
		*NewFunctionBase("greatest", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Greatest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns JSON value that is all-encompassing.
*/
func (this *Greatest) Type() value.Type { return value.JSON }

/*
Calls the Eval function and passes in the receiver, current item and
current context.
*/
func (this *Greatest) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method takes in a set of values args and context and returns a value.
Range over the args values, check the type. If it is less that or equal to
a NULL then continue, If it is a NULL_VALUE then set the greatest to that argument.
This is exercised only once. The final check is to see if Collate returns a
positive value then set the greatest to that value and return it.
*/
func (this *Greatest) Apply(context Context, args ...value.Value) (value.Value, error) {
	rv := value.NULL_VALUE
	for _, a := range args {
		if a.Type() <= value.NULL {
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
function is MaxInt16  = 1<<15 - 1. This is defined using the
math package.
*/
func (this *Greatest) MaxArgs() int { return math.MaxInt16 }

/*
The constructor returns a NewGreatest FunctionConstructor.
*/
func (this *Greatest) Constructor() FunctionConstructor { return NewGreatest }

///////////////////////////////////////////////////
//
// Least
//
///////////////////////////////////////////////////

/*
This represents the comparison function LEAST(expr1, expr2..). It is
the smallest non-NULL, non-MISSING value if the values are of the
same type, otherwise NULL. Type Least is a struct that implements
FunctionBase.
*/
type Least struct {
	FunctionBase
}

/*
The function NewLeast takes as input expressions and returns
a pointer to the Least struct that calls NewFunctionBase to
create a function named LEAST with input operands as the
expressions.
*/
func NewLeast(operands ...Expression) Function {
	rv := &Least{
		*NewFunctionBase("least", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Least) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns JSON value that is all-encompassing.
*/
func (this *Least) Type() value.Type { return value.JSON }

/*
Calls the Eval function and passes in the receiver, current item and
current context.
*/
func (this *Least) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method takes in a set of values args and context and returns a value.
Range over the args values, check the type. If it is less that or equal to
a NULL then continue, If it is a NULL_VALUE then set the least as that
argument. This is exercised only once. The final check is to see if Collate
returns a negative value then set the least to that value and return it.
*/
func (this *Least) Apply(context Context, args ...value.Value) (value.Value, error) {
	rv := value.NULL_VALUE

	for _, a := range args {
		if a.Type() <= value.NULL {
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
function is MaxInt16  = 1<<15 - 1. This is defined using the
math package.
*/
func (this *Least) MaxArgs() int { return math.MaxInt16 }

/*
The constructor returns a NewLeast FunctionConstructor.
*/
func (this *Least) Constructor() FunctionConstructor { return NewLeast }

///////////////////////////////////////////////////
//
// Successor
//
///////////////////////////////////////////////////

type Successor struct {
	UnaryFunctionBase
}

func NewSuccessor(operand Expression) Function {
	rv := &Successor{
		*NewUnaryFunctionBase("successor", operand),
	}

	rv.expr = rv
	return rv
}

func (this *Successor) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Successor) Type() value.Type {
	return this.Operand().Type().Successor()
}

func (this *Successor) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Successor) Apply(context Context, arg value.Value) (value.Value, error) {
	return arg.Successor(), nil
}

func (this *Successor) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSuccessor(operands[0])
	}
}
