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
	"fmt"
	"math"
	"strconv"

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// ToArray
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TO_ARRAY(expr).
It returns an array where a missing, null and arrays map
to themselves and all other values are wrapped in an array.
ToArray is a struct that implements UnaryFunctionBase.
*/
type ToArray struct {
	UnaryFunctionBase
}

/*
The function NewToArray takes as input an expression and returns
a pointer to the ToArray struct that calls NewUnaryFunctionBase to
create a function named TO_ARRAY with an input operand as the
expression.
*/
func NewToArray(operand Expression) Function {
	rv := &ToArray{
		*NewUnaryFunctionBase("to_array", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ToArray) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Array value.
*/
func (this *ToArray) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ToArray) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns the argument itself if type of the input value is Null,
a value below this (N!QL order) or an Array. Otherwise convert the
argument to a valid Go type ang cast it to a slice of interface.
*/
func (this *ToArray) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	} else if arg.Type() == value.ARRAY {
		return arg, nil
	}

	return value.NewValue([]interface{}{arg.Actual()}), nil
}

/*
The constructor returns a NewToArray with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *ToArray) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToArray(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToAtom
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TO_ATOM(expr).
It returns atomic values where, missing, null, boolean,
numbers and strings, are themselves, arrays of length 1
are the result of TO_ATOM() on their single element and
objects of length 1 are the result of TO_ATOM() on their
single value. All other values return null. ToAtom is a
struct that implements UnaryFunctionBase.
*/
type ToAtom struct {
	UnaryFunctionBase
}

/*
The function NewToAtom takes as input an expression and returns
a pointer to the ToAtom struct that calls NewUnaryFunctionBase to
create a function named TO_ATOM with an input operand as the
expression.
*/
func NewToAtom(operand Expression) Function {
	rv := &ToAtom{
		*NewUnaryFunctionBase("to_atom", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ToAtom) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns the type of the operand. If its type is lower than
that of the ARRAY as per type ordering defined by N!QL specs
return that type, else return a JSON value.
*/
func (this *ToAtom) Type() value.Type {
	t := this.Operand().Type()
	if t < value.ARRAY {
		return t
	} else {
		return value.JSON
	}
}

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ToAtom) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns value based on input argument type. If
the type is lower than the array in N1QL defined ordering,
then we return the argument itself. If it is an array value
and it has only one element the result of the Apply method
on this single element is returned. It it is an object
and has only one element, then the result of the Apply
method on this single value is returned. For all other cases
we return a NULL.
*/
func (this *ToAtom) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() < value.ARRAY {
		return arg, nil
	} else {
		switch a := arg.Actual().(type) {
		case []interface{}:
			if len(a) == 1 {
				return this.Apply(context, value.NewValue(a[0]))
			}
		case map[string]interface{}:
			if len(a) == 1 {
				for _, v := range a {
					return this.Apply(context, value.NewValue(v))
				}
			}
		}
	}

	return value.NULL_VALUE, nil
}

/*
The constructor returns a NewToAtom with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *ToAtom) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToAtom(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToBool
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TO_BOOL(expr).
It returns boolean values where missing, null, false map to
themselves. Numbers +0, -0 and NaN, empty strings, arrays
and objects as expr map to false. All other values are
true. ToBool is a struct that implements UnaryFunctionBase.
*/
type ToBool struct {
	UnaryFunctionBase
}

/*
The function NewToBool takes as input an expression and returns
a pointer to the ToBool struct that calls NewUnaryFunctionBase to
create a function named TO_BOOL with an input operand as the
expression.
*/
func NewToBool(operand Expression) Function {
	rv := &ToBool{
		*NewUnaryFunctionBase("to_bool", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ToBool) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Boolean value.
*/
func (this *ToBool) Type() value.Type { return value.BOOLEAN }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ToBool) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
If the input argument type is a missing, null or boolean value, it returns
itself. Check to see the Go type of the input. If it is float64, then
use the isNaN(returns if input is not a number) method defined in the math
package and make sure that it returns false and the number is not 0, return
true. If type is string, slice of interface or map which are are not empty,
return true. All other input types return NULLs.
*/
func (this *ToBool) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING, value.NULL, value.BOOLEAN:
		return arg, nil
	default:
		switch a := arg.Actual().(type) {
		case float64:
			return value.NewValue(!math.IsNaN(a) && a != 0), nil
		case string:
			return value.NewValue(len(a) > 0), nil
		case []interface{}:
			return value.NewValue(len(a) > 0), nil
		case map[string]interface{}:
			return value.NewValue(len(a) > 0), nil
		default:
			return value.NULL_VALUE, nil
		}
	}
}

/*
The constructor returns a NewToBool with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *ToBool) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToBool(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToNum
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TO_NUM(expr).
It returns number values where missing, null, and numbers
map to themselves. False is 0, true is 1, strings that
parse as numbers are those numbers and all other values
are null (For e.g. "123" is 123 but "a12" will be NULL).
ToNum is a struct that implements UnaryFunctionBase.
*/
type ToNum struct {
	UnaryFunctionBase
}

/*
The function NewToNum takes as input an expression and returns
a pointer to the ToNum struct that calls NewUnaryFunctionBase to
create a function named TO_NUM with an input operand as the
expression.
*/
func NewToNum(operand Expression) Function {
	rv := &ToNum{
		*NewUnaryFunctionBase("to_num", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ToNum) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Number value.
*/
func (this *ToNum) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ToNum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns number values where missing, null, and numbers
return themselves. If the input Go type (obtained by calling
the Actual method) is bool then, if value is false return 0,
and if true return 1. For strings use the ParseFloat method
defined in strconv to determine if the parsed string is a
valid number and return that number. For all other types
return a Null value.
*/
func (this *ToNum) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING, value.NULL, value.NUMBER:
		return arg, nil
	default:
		switch a := arg.Actual().(type) {
		case bool:
			if a {
				return value.NewValue(1.0), nil
			} else {
				return value.NewValue(0.0), nil
			}
		case string:
			f, err := strconv.ParseFloat(a, 64)
			if err == nil {
				return value.NewValue(f), nil
			}
		}
	}

	return value.NULL_VALUE, nil
}

/*
The constructor returns a NewToNum with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *ToNum) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToNum(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToObj
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TOOBJ(expr).
It returns an object value. The input of types missing,
null and object return themselves. For all other values,
return an _EMPTY_OBJECT value. ToObj is a struct that
implements UnaryFunctionBase.
*/
type ToObj struct {
	UnaryFunctionBase
}

/*
The function NewToObj takes as input an expression and returns
a pointer to the ToObj struct that calls NewUnaryFunctionBase to
create a function named TO_OBJ with an input operand as the
expression.
*/
func NewToObj(operand Expression) Function {
	rv := &ToObj{
		*NewUnaryFunctionBase("to_obj", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ToObj) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a Object value.
*/
func (this *ToObj) Type() value.Type { return value.OBJECT }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ToObj) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
Variable _EMPTY_OBJECT is a N1QL value that is a map from
string to interface. It is an OBJECT that has no entries.
*/
var _EMPTY_OBJECT = value.NewValue(map[string]interface{}{})

/*
This method returns an object value. The input of types
missing, null and object return themselves. For all other
values, return an _EMPTY_OBJECT value.
*/
func (this *ToObj) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING, value.NULL, value.OBJECT:
		return arg, nil
	}

	return _EMPTY_OBJECT, nil
}

/*
The constructor returns a NewToObj with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *ToObj) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToObj(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ToStr
//
///////////////////////////////////////////////////

/*
This represents the type conversion function TOSTR(expr).
It returns a string based on the input expr value. Values
missing, null and strings return themselves. False, true
(boolean) and numbers return their string representation.
All other values map to null. ToStr is a struct that
implements UnaryFunctionBase.
*/
type ToStr struct {
	UnaryFunctionBase
}

/*
The function NewToStr takes as input an expression and returns
a pointer to the ToStr struct that calls NewUnaryFunctionBase to
create a function named TO_STR with an input operand as the
expression.
*/
func NewToStr(operand Expression) Function {
	rv := &ToStr{
		*NewUnaryFunctionBase("to_str", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ToStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a String value.
*/
func (this *ToStr) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
It returns a string based on the input expr value. Values
missing, null and strings return themselves. False, true
(boolean) and numbers return their string representation.
This is done using the Sprint method defined in fmt for Go.
All other values map to null.
*/
func (this *ToStr) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING, value.NULL, value.STRING:
		return arg, nil
	case value.BOOLEAN, value.NUMBER:
		return value.NewValue(fmt.Sprint(arg.Actual())), nil
	default:
		return value.NULL_VALUE, nil
	}
}

/*
The constructor returns a NewToStr with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *ToStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToStr(operands[0])
	}
}
