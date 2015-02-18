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
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// IsArray
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISARRAY(expr).
It returns true if expr is an array; else false. IsArray
is a struct that implements UnaryFunctionBase.
*/
type IsArray struct {
	UnaryFunctionBase
}

/*
The function NewIsArray takes as input an expression and returns
a pointer to the IsArray struct that calls NewUnaryFunctionBase to
create a function named IS_ARRAY with an input operand as the
expression.
*/
func NewIsArray(operand Expression) Function {
	rv := &IsArray{
		*NewUnaryFunctionBase("is_array", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsArray) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsArray) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsArray) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is an array value, else false.
*/
func (this *IsArray) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING || arg.Type() == value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.ARRAY), nil
}

/*
The constructor returns a NewIsArray with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsArray) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsArray(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsAtom
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISATOM(expr).
Returns true if expr is a boolean, number, or string;
else false. IsAtom is a struct that implements
UnaryFunctionBase.
*/
type IsAtom struct {
	UnaryFunctionBase
}

/*
The function NewIsAtom takes as input an expression and returns
a pointer to the IsAtom struct that calls NewUnaryFunctionBase to
create a function named IS_ATOM with an input operand as the
expression.
*/
func NewIsAtom(operand Expression) Function {
	rv := &IsAtom{
		*NewUnaryFunctionBase("is_atom", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsAtom) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsAtom) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsAtom) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
Checks the type of input argument and returns true for boolean,
number and string and false for all other values.
*/
func (this *IsAtom) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING || arg.Type() == value.NULL {
		return arg, nil
	}

	switch arg.Type() {
	case value.BOOLEAN, value.NUMBER, value.STRING:
		return value.TRUE_VALUE, nil
	default:
		return value.FALSE_VALUE, nil
	}
}

/*
The constructor returns a NewIsAtom with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsAtom) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsAtom(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsBinary
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISBOOLEAN(expr).
Returns true if expr is a boolean; else false. IsBinary is
a struct that implements UnaryFunctionBase.
*/
type IsBinary struct {
	UnaryFunctionBase
}

/*
The function NewIsBinary takes as input an expression and returns
a pointer to the IsBinary struct that calls NewUnaryFunctionBase to
create a function named IS_BOOL with an input operand as the
expression.
*/
func NewIsBinary(operand Expression) Function {
	rv := &IsBinary{
		*NewUnaryFunctionBase("is_binary", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsBinary) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsBinary) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsBinary) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is a boolean value, else false.
*/
func (this *IsBinary) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING || arg.Type() == value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.BINARY), nil
}

/*
The constructor returns a NewIsBinary with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsBinary) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsBinary(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsBoolean
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISBOOLEAN(expr).
Returns true if expr is a boolean; else false. IsBoolean is
a struct that implements UnaryFunctionBase.
*/
type IsBoolean struct {
	UnaryFunctionBase
}

/*
The function NewIsBoolean takes as input an expression and returns
a pointer to the IsBoolean struct that calls NewUnaryFunctionBase to
create a function named IS_BOOL with an input operand as the
expression.
*/
func NewIsBoolean(operand Expression) Function {
	rv := &IsBoolean{
		*NewUnaryFunctionBase("is_boolean", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsBoolean) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsBoolean) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsBoolean) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is a boolean value, else false.
*/
func (this *IsBoolean) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING || arg.Type() == value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.BOOLEAN), nil
}

/*
The constructor returns a NewIsBoolean with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsBoolean) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsBoolean(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsNumber
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISNUMBER(expr).
Returns true if expr is a number; else false. IsNumber is
a struct that implements UnaryFunctionBase.
*/
type IsNumber struct {
	UnaryFunctionBase
}

/*
The function NewIsNumber takes as input an expression and returns
a pointer to the IsNumber struct that calls NewUnaryFunctionBase to
create a function named IS_NUMBER with an input operand as the
expression.
*/
func NewIsNumber(operand Expression) Function {
	rv := &IsNumber{
		*NewUnaryFunctionBase("is_number", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsNumber) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsNumber) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsNumber) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is an number value, else false.
*/
func (this *IsNumber) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING || arg.Type() == value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.NUMBER), nil
}

/*
The constructor returns a NewIsNumber with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsNumber) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNumber(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsObject
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISOBJECT(expr).
Returns true if expr is an object; else false. IsObject
is a struct that implements UnaryFunctionBase.
*/
type IsObject struct {
	UnaryFunctionBase
}

/*
The function NewIsObject takes as input an expression and returns
a pointer to the IsObject struct that calls NewUnaryFunctionBase to
create a function named IS_OBJECT with an input operand as the
expression.
*/
func NewIsObject(operand Expression) Function {
	rv := &IsObject{
		*NewUnaryFunctionBase("is_object", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsObject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsObject) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsObject) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is an object value, else false.
*/
func (this *IsObject) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING || arg.Type() == value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.OBJECT), nil
}

/*
The constructor returns a NewIsObject with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsObject) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsObject(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsString
//
///////////////////////////////////////////////////

/*
This represents the Type checking function ISSTRING(expr).
Returns true if expr is a string; else false. Type IsString
is a struct that implements UnaryFunctionBase.
*/
type IsString struct {
	UnaryFunctionBase
}

/*
The function NewIsString input an expression and returns
a pointer to the IsString struct that calls NewUnaryFunctionBase to
create a function named IS_STRING with an input operand as the
expression.
*/
func NewIsString(operand Expression) Function {
	rv := &IsString{
		*NewUnaryFunctionBase("is_string", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsString) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsString) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsString) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is a string value, else false.
*/
func (this *IsString) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING || arg.Type() == value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.STRING), nil
}

/*
The constructor returns a NewIsString with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsString) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsString(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Type
//
///////////////////////////////////////////////////

/*
This represents the type checking function TYPENAME(expr).
Returns the type based on the value of the expr as a string.
Type is a struct that implements UnaryFunctionBase.
*/
type Type struct {
	UnaryFunctionBase
}

/*
The function NewType takes as input an expression and returns
a pointer to the Type struct that calls NewUnaryFunctionBase to
create a function named TYPE_NAME with an input operand as the
expression.
*/
func NewType(operand Expression) Function {
	rv := &Type{
		*NewUnaryFunctionBase("type", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Type) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a String value.
*/
func (this *Type) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Type) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns the type of the input value as a string value.
*/
func (this *Type) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type().String()), nil
}

/*
The constructor returns a NewType with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Type) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewType(operands[0])
	}
}
