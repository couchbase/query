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
	"encoding/json"

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// DecodeJSON
//
///////////////////////////////////////////////////

/*
This represents the  json function DECODE_JSON(expr). It
unmarshals the JSON-encoded string into a N1QL value, and
if empty string is MISSING. Type DecodeJSON is a struct
that implements UnaryFunctionBase.
*/
type DecodeJSON struct {
	UnaryFunctionBase
}

/*
The function NewDecodeJSON calls NewUnaryFunctionBase to
create a function named DECODE_JSON with an expression as
input.
*/
func NewDecodeJSON(operand Expression) Function {
	rv := &DecodeJSON{
		*NewUnaryFunctionBase("decode_json", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *DecodeJSON) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type JSON.
*/
func (this *DecodeJSON) Type() value.Type { return value.JSON }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *DecodeJSON) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns a valid N1QL value from a JSON encoded
string. If the input type is missing return missing, and if
it isnt string then return null value. Conver the input arg
to valid Go type and cast to a string. If it is an empty
string return missing value. If not then call the Unmarshal
method defined in the json package, by casting the strings
to a bytes slice and return the json value.
*/
func (this *DecodeJSON) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.Actual().(string)
	if s == "" {
		return value.MISSING_VALUE, nil
	}

	var p interface{}
	err := json.Unmarshal([]byte(s), &p)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(p), nil
}

/*
The constructor returns a NewDecodeJSON with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *DecodeJSON) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDecodeJSON(operands[0])
	}
}

///////////////////////////////////////////////////
//
// EncodeJSON
//
///////////////////////////////////////////////////

/*
This represents the  json function ENCODE_JSON(expr).
It marshals the N1QL value into a JSON-encoded string.
A MISSING becomes the empty string. Type EncodeJSON
is a struct that implements UnaryFunctionBase.
*/
type EncodeJSON struct {
	UnaryFunctionBase
}

/*
The function NewEncodeJSON calls NewUnaryFunctionBase to
create a function named ENCODE_JSON with an expression as
input.
*/
func NewEncodeJSON(operand Expression) Function {
	rv := &EncodeJSON{
		*NewUnaryFunctionBase("encode_json", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *EncodeJSON) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type STRING.
*/
func (this *EncodeJSON) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *EncodeJSON) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns a Json encoded string by sing the MarshalJSON
method. The return bytes value is cast to a string and returned.
*/
func (this *EncodeJSON) Apply(context Context, arg value.Value) (value.Value, error) {
	bytes, _ := arg.MarshalJSON()
	return value.NewValue(string(bytes)), nil
}

/*
The constructor returns a NewEncodeJSON with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *EncodeJSON) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewEncodeJSON(operands[0])
	}
}

///////////////////////////////////////////////////
//
// EncodedSize
//
///////////////////////////////////////////////////

/*
This represents the json function ENCODED_SIZE(expr). It
returns a number of bytes in an uncompressed JSON encoding
of the value. The exact size is implementation-dependent.
Always returns an integer, and never MISSING or NULL;
returns 0 for MISSING.
*/
type EncodedSize struct {
	UnaryFunctionBase
}

/*
The function NewEncodedSize calls NewUnaryFunctionBase to
create a function named ENCODED_SIZE with an expression as
input.
*/
func NewEncodedSize(operand Expression) Function {
	rv := &EncodedSize{
		*NewUnaryFunctionBase("encoded_size", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *EncodedSize) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *EncodedSize) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *EncodedSize) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns a number value that represents the length of the bytes slice
returned by the MarshalJSON method cast to a float64 value.
*/
func (this *EncodedSize) Apply(context Context, arg value.Value) (value.Value, error) {
	bytes, _ := arg.MarshalJSON()
	return value.NewValue(float64(len(bytes))), nil
}

/*
The constructor returns a NewEncodedSize with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *EncodedSize) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewEncodedSize(operands[0])
	}
}

///////////////////////////////////////////////////
//
// PolyLength
//
///////////////////////////////////////////////////

/*
This represents the json function POLY_LENGTH(expr).
It returns the length of the value after evaluating
the expression. The exact meaning of length depends
on the type of the value. For missing, null it returns
a missing and null, for a string it returns the length
of the string, for array it returns the number of
elements, for objects it returns the number of
name/value pairs in the object and for any other value
it returns a NULL. Type PolyLength is a struct that
implements UnaryFunctionBase.
*/
type PolyLength struct {
	UnaryFunctionBase
}

/*
The function NewPolyLength calls NewUnaryFunctionBase to
create a function named POLY_LENGTH with an expression as
input.
*/
func NewPolyLength(operand Expression) Function {
	rv := &PolyLength{
		*NewUnaryFunctionBase("poly_length", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *PolyLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *PolyLength) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *PolyLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method evaluates the input value and returns the length
based on its type. If the input argument is a missing then
return a missing value. Convert it to a valid Go type. If
it is a string slice of interfaces or object then return
its length cast as a number float64. By default return a
null value.
*/
func (this *PolyLength) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	switch oa := arg.Actual().(type) {
	case string:
		return value.NewValue(float64(len(oa))), nil
	case []interface{}:
		return value.NewValue(float64(len(oa))), nil
	case map[string]interface{}:
		return value.NewValue(float64(len(oa))), nil
	default:
		return value.NULL_VALUE, nil
	}
}

/*
The constructor returns a NewPolyLength with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *PolyLength) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPolyLength(operands[0])
	}
}
