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
	"github.com/couchbaselabs/query/value"
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
	switch arg.Type() {
	case value.BOOLEAN, value.NUMBER, value.STRING:
		return value.NewValue(true), nil
	default:
		return value.NewValue(false), nil
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
// IsBool
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISBOOL(expr).
Returns true if expr is a boolean; else false. IsBool is
a struct that implements UnaryFunctionBase.
*/
type IsBool struct {
	UnaryFunctionBase
}

/*
The function NewIsBool takes as input an expression and returns
a pointer to the IsBool struct that calls NewUnaryFunctionBase to
create a function named IS_BOOL with an input operand as the
expression.
*/
func NewIsBool(operand Expression) Function {
	rv := &IsBool{
		*NewUnaryFunctionBase("is_bool", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsBool) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsBool) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsBool) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is a boolean value, else false.
*/
func (this *IsBool) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.BOOLEAN), nil
}

/*
The constructor returns a NewIsBool with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsBool) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsBool(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsNum
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISNUM(expr).
Returns true if expr is a number; else false. IsNum is
a struct that implements UnaryFunctionBase.
*/
type IsNum struct {
	UnaryFunctionBase
}

/*
The function NewIsNum takes as input an expression and returns
a pointer to the IsNum struct that calls NewUnaryFunctionBase to
create a function named IS_NUM with an input operand as the
expression.
*/
func NewIsNum(operand Expression) Function {
	rv := &IsNum{
		*NewUnaryFunctionBase("is_num", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsNum) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsNum) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsNum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is an number value, else false.
*/
func (this *IsNum) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.NUMBER), nil
}

/*
The constructor returns a NewIsNum with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsNum) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNum(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsObj
//
///////////////////////////////////////////////////

/*
This represents the type checking function ISOBJ(expr).
Returns true if expr is an object; else false. IsObj
is a struct that implements UnaryFunctionBase.
*/
type IsObj struct {
	UnaryFunctionBase
}

/*
The function NewIsObj takes as input an expression and returns
a pointer to the IsObj struct that calls NewUnaryFunctionBase to
create a function named IS_OBJ with an input operand as the
expression.
*/
func NewIsObj(operand Expression) Function {
	rv := &IsObj{
		*NewUnaryFunctionBase("is_obj", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsObj) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsObj) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsObj) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is an object value, else false.
*/
func (this *IsObj) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.OBJECT), nil
}

/*
The constructor returns a NewIsObj with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsObj) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsObj(operands[0])
	}
}

///////////////////////////////////////////////////
//
// IsStr
//
///////////////////////////////////////////////////

/*
This represents the Type checking function ISSTR(expr).
Returns true if expr is a string; else false. Type IsStr
is a struct that implements UnaryFunctionBase.
*/
type IsStr struct {
	UnaryFunctionBase
}

/*
The function NewIsStr input an expression and returns
a pointer to the IsStr struct that calls NewUnaryFunctionBase to
create a function named IS_STR with an input operand as the
expression.
*/
func NewIsStr(operand Expression) Function {
	rv := &IsStr{
		*NewUnaryFunctionBase("is_str", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *IsStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *IsStr) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *IsStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns true if type of the input value is a string value, else false.
*/
func (this *IsStr) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.STRING), nil
}

/*
The constructor returns a NewIsStr with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *IsStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsStr(operands[0])
	}
}

///////////////////////////////////////////////////
//
// TypeName
//
///////////////////////////////////////////////////

/*
This represents the type checking function TYPENAME(expr).
Returns the type based on the value of the expr as a string.
TypeName is a struct that implements UnaryFunctionBase.
*/
type TypeName struct {
	UnaryFunctionBase
}

/*
The function NewTypeName takes as input an expression and returns
a pointer to the TypeName struct that calls NewUnaryFunctionBase to
create a function named TYPE_NAME with an input operand as the
expression.
*/
func NewTypeName(operand Expression) Function {
	rv := &TypeName{
		*NewUnaryFunctionBase("type_name", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *TypeName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a String value.
*/
func (this *TypeName) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *TypeName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns the type of the input value as a string value.
*/
func (this *TypeName) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type().String()), nil
}

/*
The constructor returns a NewTypeName with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *TypeName) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewTypeName(operands[0])
	}
}
