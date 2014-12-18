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
	"math/rand"

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// Abs
//
///////////////////////////////////////////////////

/*
This represents the number function ABS(expr). It returns
the absolute value of the number. Type Abs is a struct that
implements UnaryFunctionBase.
*/
type Abs struct {
	UnaryFunctionBase
}

/*
The function NewAbs takes as input an expression and returns
a pointer to the Abs struct that calls NewUnaryFunctionBase to
create a function named ABS with an input operand as the
expression.
*/
func NewAbs(operand Expression) Function {
	rv := &Abs{
		*NewUnaryFunctionBase("abs", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Abs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Abs) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Abs) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns an absolute
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Abs method, and cast to float64. Return the new value.
*/
func (this *Abs) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Abs(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewAbs with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Abs) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAbs(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Acos
//
///////////////////////////////////////////////////

/*
This represents the number function ACOS(expr). It returns
the arccosine in radians of the input value. Type Acos is
a struct that implements UnaryFunctionBase.
*/
type Acos struct {
	UnaryFunctionBase
}

/*
The function NewAcos takes as input an expression and returns
a pointer to the Acos struct that calls NewUnaryFunctionBase to
create a function named ACOS with an input operand as the
expression.
*/
func NewAcos(operand Expression) Function {
	rv := &Acos{
		*NewUnaryFunctionBase("acos", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Acos) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Acos) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Acos) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its acos
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Acos method, and cast to float64. Return the new value.
*/
func (this *Acos) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Acos(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewAcos with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Acos) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAcos(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Asin
//
///////////////////////////////////////////////////

/*
This represents the number function ASIN(expr). It returns
the arcsine in radians of the input value. Type Asin is
a struct that implements UnaryFunctionBase.
*/
type Asin struct {
	UnaryFunctionBase
}

/*
The function NewAsin takes as input an expression and returns
a pointer to the Asin struct that calls NewUnaryFunctionBase to
create a function named ASIN with an input operand as the
expression.
*/
func NewAsin(operand Expression) Function {
	rv := &Asin{
		*NewUnaryFunctionBase("asin", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Asin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Asin) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Asin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns the Asin
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Asin method, and cast to float64. Return the new value.
*/
func (this *Asin) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Asin(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewAsin with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Asin) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAsin(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Atan
//
///////////////////////////////////////////////////

/*
This represents the number function ATAN(expr). It returns
the arctangent in radians of the input value. Type Atan is
a struct that implements UnaryFunctionBase.
*/
type Atan struct {
	UnaryFunctionBase
}

/*
The function NewAtan takes as input an expression and returns
a pointer to the Atan struct that calls NewUnaryFunctionBase to
create a function named ATAN with an input operand as the
expression.
*/
func NewAtan(operand Expression) Function {
	rv := &Atan{
		*NewUnaryFunctionBase("atan", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Atan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Atan) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Atan) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its arctangent
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Atan method, and cast to float64. Return the new value.
*/
func (this *Atan) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Atan(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewAtan with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Atan) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAtan(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Atan2
//
///////////////////////////////////////////////////

/*
This represents the number function ATAN2(expr1, expr2).
It returns the arctangent of expr2/expr1. Type Atan2 is
a struct that implements BinaryFunctionBase.
*/
type Atan2 struct {
	BinaryFunctionBase
}

/*
The function NewAtan2 calls NewBinaryFunctionBase to
create a function named ATAN2 with the two
expressions as input.
*/
func NewAtan2(first, second Expression) Function {
	rv := &Atan2{
		*NewBinaryFunctionBase("atan2", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Atan2) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Atan2) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *Atan2) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method evaluates the atan2 value of the input expressions. If either
of the input argument types are missing, or not a number return a missing
and null value respectively. For numbers, use the Atan2 method defined by
the math package with the first and second values as input and return.
*/
func (this *Atan2) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Atan2(
		first.Actual().(float64),
		second.Actual().(float64))), nil
}

/*
The constructor returns a NewAtan2 with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *Atan2) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAtan2(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Ceil
//
///////////////////////////////////////////////////

/*
This represents the number function CEIL(expr). It
represents the smallest integer not less than the
number. Type Ceil is a struct that implements
UnaryFunctionBase.
*/
type Ceil struct {
	UnaryFunctionBase
}

/*
The function NewCeil takes as input an expression and returns
a pointer to the Ceil struct that calls NewUnaryFunctionBase to
create a function named CEIL with an input operand as the
expression.
*/
func NewCeil(operand Expression) Function {
	rv := &Ceil{
		*NewUnaryFunctionBase("ceil", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Ceil) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Ceil) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Ceil) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns the int
value not less than the input. If the type of operand is missing then
return it. For values that are not of type Number, return a null
value. For numbers use the math package Ceil method, and cast to
float64. Return the new value.
*/
func (this *Ceil) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Ceil(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewCeil with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Ceil) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewCeil(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Cos
//
///////////////////////////////////////////////////

/*
This represents the number function COS(expr). It returns
the cosine of the input value. Type Cos is
a struct that implements UnaryFunctionBase.
*/
type Cos struct {
	UnaryFunctionBase
}

/*
The function NewCos takes as input an expression and returns
a pointer to the Cos struct that calls NewUnaryFunctionBase to
create a function named COS with an input operand as the
expression.
*/
func NewCos(operand Expression) Function {
	rv := &Cos{
		*NewUnaryFunctionBase("cos", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Cos) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Cos) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Cos) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its cos
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package cos method, and cast to float64. Return the new value.
*/
func (this *Cos) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Cos(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewCos with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Cos) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewCos(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Degrees
//
///////////////////////////////////////////////////

/*
This represents the number function DEGREES(expr). It
converts input radians to degrees. Type Degrees is a
struct that implements UnaryFunctionBase.
*/
type Degrees struct {
	UnaryFunctionBase
}

/*
The function NewDegrees takes as input an expression and returns
a pointer to the Degrees struct that calls NewUnaryFunctionBase to
create a function named DEGREES with an input operand as the
expression.
*/
func NewDegrees(operand Expression) Function {
	rv := &Degrees{
		*NewUnaryFunctionBase("degrees", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Degrees) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Degrees) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Degrees) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and converts it from
radians to degrees. If the type of operand is missing then return it.
For values that are not of type Number, return a null value. For
numbers use ( a* 180 / math.PI). Return the value.
*/
func (this *Degrees) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(arg.Actual().(float64) * 180.0 / math.Pi), nil
}

/*
The constructor returns a NewDegrees with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Degrees) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDegrees(operands[0])
	}
}

///////////////////////////////////////////////////
//
// E
//
///////////////////////////////////////////////////

/*
This represents the number function E(). It
returns eulers number which is used as a base
of natural logarithms. Type E is a struct that
implements NullaryFunctionBase.
*/
type E struct {
	NullaryFunctionBase
}

var _E = NewE()

/*
The function NewE returns a pointer to the
NewNullaryFunctionBase to create a function E. It has
no input arguments.
*/
func NewE() Function {
	rv := &E{
		*NewNullaryFunctionBase("e"),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *E) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *E) Type() value.Type { return value.NUMBER }

/*
Returns _E_VALUE.
*/
func (this *E) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _E_VALUE, nil
}

/*
Returns _E_VALUE.
*/
func (this *E) Value() value.Value {
	return _E_VALUE
}

/*
Return receiver as FunctionConstructor.
*/
func (this *E) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _E }
}

/*
Variable _E_VALUE uses the math package to define
the Euler number used as a base in logs.
*/
var _E_VALUE = value.NewValue(math.E)

///////////////////////////////////////////////////
//
// Exp
//
///////////////////////////////////////////////////

/*
This represents the number function EXP(expr). It
represents e to the power expr. Type Exp is a struct
that implements UnaryFunctionBase.
*/
type Exp struct {
	UnaryFunctionBase
}

/*
The function NewExp takes as input an expression and returns
a pointer to the Exp struct that calls NewUnaryFunctionBase to
create a function named EXP with an input operand as the
expression.
*/
func NewExp(operand Expression) Function {
	rv := &Exp{
		*NewUnaryFunctionBase("exp", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Exp) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Exp) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Exp) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its exp
value wrt E. If the type of operand is missing then return it. For
values that are not of type Number, return a null value. For numbers
use the math package Exp method, and cast to float64. Return the new value.
*/
func (this *Exp) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Exp(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewExp with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Exp) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewExp(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Ln
//
///////////////////////////////////////////////////

/*
This represents the number function LN(expr). It
computes log base e. Type Ln is a struct that
implements UnaryFunctionBase.
*/
type Ln struct {
	UnaryFunctionBase
}

/*
The function NewLn takes as input an expression and returns
a pointer to the Ln struct that calls NewUnaryFunctionBase to
create a function named LN with an input operand as the
expression.
*/
func NewLn(operand Expression) Function {
	rv := &Ln{
		*NewUnaryFunctionBase("ln", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Ln) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Ln) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Ln) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its Ln
value. If the type of operand is missing then return it. For
values that are not of type Number, return a null value. For numbers
use the math package Log method, and cast to float64. Return the new value.
*/
func (this *Ln) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Log(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewLn with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Ln) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLn(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Log
//
///////////////////////////////////////////////////

/*
This represents the number function LOG(expr). It
computes log base 10. Type Log is a struct that
implements UnaryFunctionBase.
*/
type Log struct {
	UnaryFunctionBase
}

/*
The function NewLog takes as input an expression and returns
a pointer to the Log struct that calls NewUnaryFunctionBase to
create a function named LOG with an input operand as the
expression.
*/
func NewLog(operand Expression) Function {
	rv := &Log{
		*NewUnaryFunctionBase("log", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Log) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Log) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Log) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its log
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Log10 method, and cast to float64. Return the new value.
(Log to the base 10 value)
*/
func (this *Log) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Log10(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewLog with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Log) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLog(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Floor
//
///////////////////////////////////////////////////

/*
This represents the number function FLOOR(expr). It
returns the largest integer not greater than the
number. Type Floor is a struct that implements
UnaryFunctionBase.
*/
type Floor struct {
	UnaryFunctionBase
}

/*
The function NewFloor takes as input an expression and returns
a pointer to the Floor struct that calls NewUnaryFunctionBase to
create a function named FLOOR with an input operand as the
expression.
*/
func NewFloor(operand Expression) Function {
	rv := &Floor{
		*NewUnaryFunctionBase("floor", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Floor) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Floor) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Floor) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns the floor
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Floor method, and cast to float64. Return the new value.
*/
func (this *Floor) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Floor(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewFloor with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Floor) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewFloor(operands[0])
	}
}

///////////////////////////////////////////////////
//
// NaN
//
///////////////////////////////////////////////////

/*
This represents the number function NaN(). It returns
an IEEE 754 “not-a-number” value. Type NaN is a struct
that implements NullaryFunctionBase.
*/
type NaN struct {
	NullaryFunctionBase
}

var _NAN = NewNaN()

/*
The function NewNaN returns a pointer to the
NewNullaryFunctionBase to create a function NAN. It has
no input arguments.
*/
func NewNaN() Function {
	rv := &NaN{
		*NewNullaryFunctionBase("nan"),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *NaN) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *NaN) Type() value.Type { return value.NUMBER }

/*
Return _NAN_VALUE.
*/
func (this *NaN) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _NAN_VALUE, nil
}

/*
Return _NAN_VALUE.
*/
func (this *NaN) Value() value.Value {
	return _NAN_VALUE
}

/*
Return method receiver as FunctionConstructor.
*/
func (this *NaN) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _NAN }
}

/*
Var _NAN_VALUE uses the math package NaN method to
define an IEEE 754 “not-a-number” value.
*/
var _NAN_VALUE = value.NewValue(math.NaN())

///////////////////////////////////////////////////
//
// NegInf
//
///////////////////////////////////////////////////

/*
This represents the number function NEGINF(). It
returns the negative infinity number value. Type
NegInf is a struct that implements NullaryFunctionBase.
*/
type NegInf struct {
	NullaryFunctionBase
}

var _NEG_INF = NewNegInf()

/*
The function NewNegInf returns a pointer to the
NewNullaryFunctionBase to create a function NEGINF. It has
no input arguments.
*/
func NewNegInf() Function {
	rv := &NegInf{
		*NewNullaryFunctionBase("neginf"),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *NegInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *NegInf) Type() value.Type { return value.NUMBER }

/*
Returns _NEGINF_VALUE.
*/
func (this *NegInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _NEGINF_VALUE, nil
}

/*
Returns _NEGINF_VALUE.
*/
func (this *NegInf) Value() value.Value {
	return _NEGINF_VALUE
}

/*
Return method receiver as FunctionConstructor.
*/
func (this *NegInf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _NEG_INF }
}

/*
Variable _NEGINF_VALUE the negative infinity value
defined using the method Inf(-1) in the  math package.
*/
var _NEGINF_VALUE = value.NewValue(math.Inf(-1))

///////////////////////////////////////////////////
//
// PI
//
///////////////////////////////////////////////////

/*
This represents the number function PI(). It returns
PI. Type PI is a struct that implements NullaryFunctionBase.
*/
type PI struct {
	NullaryFunctionBase
}

var _PI = NewPI()

/*
The function NewPI returns a pointer to the
NewNullaryFunctionBase to create a function PI. It has
no input arguments.
*/
func NewPI() Function {
	rv := &PI{
		*NewNullaryFunctionBase("pi"),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *PI) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *PI) Type() value.Type { return value.NUMBER }

/*
Return _PI_VALUE.
*/
func (this *PI) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _PI_VALUE, nil
}

/*
Return _PI_VALUE.
*/
func (this *PI) Value() value.Value {
	return _PI_VALUE
}

/*
Return method receiver as FunctionConstructor.
*/
func (this *PI) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _PI }
}

/*
Variable _PI_VALUE uses the math package to define PI.
*/
var _PI_VALUE = value.NewValue(math.Pi)

///////////////////////////////////////////////////
//
// PosInf
//
///////////////////////////////////////////////////

/*
This represents the number function POSINF(). It
returns the positive infinity number value. Type
PosInf is a struct that implements NullaryFunctionBase.
*/
type PosInf struct {
	NullaryFunctionBase
}

var _POS_INF = NewPosInf()

/*
The function NewPosInf returns a pointer to the
NewNullaryFunctionBase to create a function POSINF. It has
no input arguments.
*/
func NewPosInf() Function {
	rv := &PosInf{
		*NewNullaryFunctionBase("posinf"),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *PosInf) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *PosInf) Type() value.Type { return value.NUMBER }

/*
Returns _POSINF_VALUE.
*/
func (this *PosInf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _POSINF_VALUE, nil
}

/*
Returns _POSINF_VALUE.
*/
func (this *PosInf) Value() value.Value {
	return _POSINF_VALUE
}

/*
Return method receiver as FunctionConstructor.
*/
func (this *PosInf) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return _POS_INF }
}

/*
Variable _POSINF_VALUE the negative infinity value
defined using the method Inf(1) in the math package.
*/
var _POSINF_VALUE = value.NewValue(math.Inf(1))

///////////////////////////////////////////////////
//
// Power
//
///////////////////////////////////////////////////

/*
This represents the number function POWER(expr1, expr2).
It returns expr1 to the power of expr2. Type Power is a
struct that implements BinaryFunctionBase.
*/
type Power struct {
	BinaryFunctionBase
}

/*
The function NewPower calls NewBinaryFunctionBase to
create a function named POWER with the two
expressions as input.
*/
func NewPower(first, second Expression) Function {
	rv := &Power{
		*NewBinaryFunctionBase("power", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Power) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Power) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *Power) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes in a context, two operand values, first and second,
and returns first^second. If type of either operand is a missing then
return missing. For values that are not of type Number, return a null
value. For numbers use the math package Pow method, and return the
value.
*/
func (this *Power) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.NUMBER || second.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Pow(
		first.Actual().(float64),
		second.Actual().(float64))), nil
}

/*
The constructor returns a NewPower with the two operands
cast to a Function as the FunctionConstructor.
*/
func (this *Power) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPower(operands[0], operands[1])
	}
}

///////////////////////////////////////////////////
//
// Radians
//
///////////////////////////////////////////////////

/*
This represents the number function RADIANS(expr).
It converts degrees to radians. Type Radians is a
struct that implements UnaryFunctionBase.
*/
type Radians struct {
	UnaryFunctionBase
}

/*
The function NewRadians takes as input an expression and returns
a pointer to the Radians struct that calls NewUnaryFunctionBase to
create a function named RADIANS with an input operand as the
expression.
*/
func NewRadians(operand Expression) Function {
	rv := &Radians{
		*NewUnaryFunctionBase("radians", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Radians) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Radians) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Radians) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and converts from degree
to radians. If the type of operand is missing then return missing. For
values that are not of type Number, return a null value. For numbers use
the formula (a * math.PI / 180.0). Return the new value.
*/
func (this *Radians) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(arg.Actual().(float64) * math.Pi / 180.0), nil
}

/*
The constructor returns a NewRadians with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Radians) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewRadians(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Random
//
///////////////////////////////////////////////////

/*
This represents the number function ROUND(expr [, digits ]).
It rounds the value to the given number of integer digits to
the right of the decimal point (left if digits is negative).
digits is 0 if not given. Type Random is a struct that
implements FunctionBase. It has a Field gen that represents
a source of random numbers as defined in the math/rand
package.
*/
type Random struct {
	FunctionBase
	gen *rand.Rand
}

/*
The method NewRandom calls NewFunctionBase to
create a function named RANDOM with input
arguments as the operands from the input expression.
For this set volatile to true. If there are no
input args, then return. If not, check the the
type of the first operand. If it is a constant,
check the operand value type. For float64 (N1QL
valid numbers) create and seed the field gen
(random number), using the rand package methods
with the seed value cast to int64. Return.
*/
func NewRandom(operands ...Expression) Function {
	rv := &Random{
		*NewFunctionBase("random", operands...),
		nil,
	}

	rv.volatile = true
	rv.expr = rv

	if len(operands) < 1 {
		return rv
	}

	switch op := operands[0].(type) {
	case *Constant:
		switch val := op.Value().Actual().(type) {
		case float64:
			rv.gen = rand.New(rand.NewSource(int64(val)))
		}
	}

	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Random) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Random) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for functions and passes in the
receiver, current item and current context.
*/
func (this *Random) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method evaluates the Random function. If the seed exists, then
return it as a value. If there are no input arguments, return
a random number. If the input argument type is Missing, return a
missing value, and if it is not a number then return a null value.
For numbers, check if it is an integer value and if not return
null. Generate a new random number using this integer value as a
seed, and return it.
*/
func (this *Random) Apply(context Context, args ...value.Value) (value.Value, error) {
	if this.gen != nil {
		return value.NewValue(this.gen.Float64()), nil
	}

	if len(args) == 0 {
		return value.NewValue(rand.Float64()), nil
	}

	arg := args[0]

	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	v := arg.Actual().(float64)
	if v != math.Trunc(v) {
		return value.NULL_VALUE, nil
	}

	gen := rand.New(rand.NewSource(int64(v)))
	return value.NewValue(gen.Float64()), nil
}

/*
Minimum input arguments required is 0.
*/
func (this *Random) MinArgs() int { return 0 }

/*
Maximum input arguments allowed is 1.
*/
func (this *Random) MaxArgs() int { return 1 }

/*
Return NewRandom as FunctionConstructor.
*/
func (this *Random) Constructor() FunctionConstructor { return NewRandom }

///////////////////////////////////////////////////
//
// Round
//
///////////////////////////////////////////////////

/*
This represents the number function ROUND(expr [, digits ]).
It rounds the value to the given number of integer digits
to the right of the decimal point (left if digits is
negative). digits is 0 if not given. Type Round is a struct
that implements FunctionBase.
*/
type Round struct {
	FunctionBase
}

/*
The function NewRound calls NewFunctionBase to
create a function named ROUND with input
arguments as the operands from the input expression.
*/
func NewRound(operands ...Expression) Function {
	rv := &Round{
		*NewFunctionBase("round", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Round) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Round) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for functions and passes in the
receiver, current item and current context.
*/
func (this *Round) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method takes in operand values and a context and rounds it of
to the given number of digits to the right or left of the decimal
point. If the input arg is missing or not a number, return a
missing value or a null value respectively. Round the value of
the input value using the method roundFloat. If it has more
than one arg then check the type. Again if missing or not a
number return a missing value/null value respectively. Make
sure this precision value is an absolute integer value. Call
the roundFloat method using the precision and return.
*/
func (this *Round) Apply(context Context, args ...value.Value) (value.Value, error) {
	arg := args[0]
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	v := arg.Actual().(float64)

	if len(this.operands) == 1 {
		return value.NewValue(roundFloat(v, 0)), nil
	}

	p := 0
	prec := args[1]
	if prec.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if prec.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	} else {
		pf := prec.Actual().(float64)
		if pf != math.Trunc(pf) {
			return value.NULL_VALUE, nil
		}
		p = int(pf)
	}

	return value.NewValue(roundFloat(v, p)), nil
}

/*
Minimum input arguments required is 1.
*/
func (this *Round) MinArgs() int { return 1 }

/*
Maximum input arguments allowed is 2.
*/
func (this *Round) MaxArgs() int { return 2 }

/*
Return NewRound as FunctionConstructor.
*/
func (this *Round) Constructor() FunctionConstructor { return NewRound }

///////////////////////////////////////////////////
//
// Sign
//
///////////////////////////////////////////////////

/*
This represents the number function SIGN(expr). It returns
-1, 0, or 1 for negative, zero, or positive numbers
respectively. Type Sign is a struct that implements
UnaryFunctionBase.
*/
type Sign struct {
	UnaryFunctionBase
}

/*
The function NewSign takes as input an expression and returns
a pointer to the Sign struct that calls NewUnaryFunctionBase to
create a function named SIGN with an input operand as the
expression.
*/
func NewSign(operand Expression) Function {
	rv := &Sign{
		*NewUnaryFunctionBase("sign", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Sign) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Sign) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Sign) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns the
sign of the value. If the type of operand is missing then return
missing. For values that are not of type Number, return a null
value. For numbers, compare to 0.0; if smaller return -1, equal
return 0 and greater return 1.
*/
func (this *Sign) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	f := arg.Actual().(float64)
	s := 0.0
	if f < 0.0 {
		s = -1.0
	} else if f > 0.0 {
		s = 1.0
	}

	return value.NewValue(s), nil
}

/*
The constructor returns a NewSign with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Sign) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSign(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Sin
//
///////////////////////////////////////////////////

/*
This represents the number function SIN(expr). It
returns the sine of the input number value. Type
Sin is a struct that implements UnaryFunctionBase.
*/
type Sin struct {
	UnaryFunctionBase
}

/*
The function NewSin takes as input an expression and returns
a pointer to the Sin struct that calls NewUnaryFunctionBase to
create a function named SIN with an input operand as the
expression.
*/
func NewSin(operand Expression) Function {
	rv := &Sin{
		*NewUnaryFunctionBase("sin", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Sin) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Sin) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Sin) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its sin
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Sin method. Return the new value.
*/
func (this *Sin) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sin(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewSin with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Sin) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSin(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Sqrt
//
///////////////////////////////////////////////////

/*
This represents the number function SQRT(expr). It returns
the square root of the input. Type Sqrt is a struct that
implements UnaryFunctionBase.
*/
type Sqrt struct {
	UnaryFunctionBase
}

/*
The function NewSqrt takes as input an expression and returns
a pointer to the Sqrt struct that calls NewUnaryFunctionBase to
create a function named SQRT with an input operand as the
expression.
*/
func NewSqrt(operand Expression) Function {
	rv := &Sqrt{
		*NewUnaryFunctionBase("sqrt", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Sqrt) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Sqrt) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Sqrt) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its sqrt
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Sqrt method. Return the new value.
*/
func (this *Sqrt) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Sqrt(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewSqrt with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Sqrt) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSqrt(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Tan
//
///////////////////////////////////////////////////

/*
This represents the number function TAN(expr). It returns
the tangent of the input. Type Tan is a struct that
implements UnaryFunctionBase.
*/
type Tan struct {
	UnaryFunctionBase
}

/*
The function NewTan takes as input an expression and returns
a pointer to the Tan struct that calls NewUnaryFunctionBase to
create a function named TAN with an input operand as the
expression.
*/
func NewTan(operand Expression) Function {
	rv := &Tan{
		*NewUnaryFunctionBase("tan", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Tan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Tan) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Tan) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns its Tan
value. If the type of operand is missing then return it. For values that
are not of type Number, return a null value. For numbers use the math
package Tan method. Return the new value.
*/
func (this *Tan) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(math.Tan(arg.Actual().(float64))), nil
}

/*
The constructor returns a NewTan with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Tan) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewTan(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Trunc
//
///////////////////////////////////////////////////

/*
This represents the number function TRUNC(expr [, digits ]).
It truncates the number to the given number of integer
digits to the right of the decimal point (left if digits is
negative). digits is 0 if not given. Type Trunc is a struct
that implements FunctionBase.
*/
type Trunc struct {
	FunctionBase
}

/*
The function NewTrunc calls NewFunctionBase to create a
function named TRUNC with input arguments as the operands
from the input expression.
*/
func NewTrunc(operands ...Expression) Function {
	rv := &Trunc{
		*NewFunctionBase("trunc", operands...),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Trunc) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *Trunc) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for functions and passes in the
receiver, current item and current context.
*/
func (this *Trunc) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

/*
This method evaluates the trunc function to truncate the given
number of integer digits to the right or left of the decimal.
If the input args (precision if given) is missing or not a
number, return a missing value or a null value respectively.
Use the truncFloat method with either 0 or input value
precision(if given) and return the value.
*/
func (this *Trunc) Apply(context Context, args ...value.Value) (value.Value, error) {
	arg := args[0]
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	}

	v := arg.Actual().(float64)

	if len(this.operands) == 1 {
		return value.NewValue(truncateFloat(v, 0)), nil
	}

	p := 0
	prec := args[1]
	if prec.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if prec.Type() != value.NUMBER {
		return value.NULL_VALUE, nil
	} else {
		pf := prec.Actual().(float64)
		if pf != math.Trunc(pf) {
			return value.NULL_VALUE, nil
		}
		p = int(pf)
	}

	return value.NewValue(truncateFloat(v, p)), nil
}

/*
Minimum input arguments required is 1.
*/
func (this *Trunc) MinArgs() int { return 1 }

/*
Maximum input arguments allowed is 2.
*/
func (this *Trunc) MaxArgs() int { return 2 }

/*
Return NewTrunc as FunctionConstructor.
*/
func (this *Trunc) Constructor() FunctionConstructor { return NewTrunc }

/*
This method is used to truncate input number to either
side of the decimal point using the integer precision.
A negative precision allows truncate to the left of
the decimal point.
*/
func truncateFloat(x float64, prec int) float64 {
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	rounder := math.Trunc(intermed)
	return rounder / pow
}

/*
This method is used to round the input number to the
given number of precision digits to the right or left
of the decimal point depending on whether it is
positive or negative. For the fraction 0.5
round towards the even value.
*/
func roundFloat(x float64, prec int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}

	sign := 1.0
	if x < 0 {
		sign = -1.0
		x = -x
	}

	pow := math.Pow(10, float64(prec))
	intermed := (x * pow) + 0.5
	rounder := math.Floor(intermed)

	// For frac 0.5, round towards even
	if rounder == intermed && math.Mod(rounder, 2) != 0 {
		rounder--
	}

	return sign * rounder / pow
}
