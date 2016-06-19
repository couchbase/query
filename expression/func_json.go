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
	"strings"

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// JSONDecode
//
///////////////////////////////////////////////////

/*
This represents the  json function JSON_DECODE(expr). It
unmarshals the JSON-encoded string into a N1QL value, and
if empty string is MISSING.
*/
type JSONDecode struct {
	UnaryFunctionBase
}

func NewJSONDecode(operand Expression) Function {
	rv := &JSONDecode{
		*NewUnaryFunctionBase("json_decode", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *JSONDecode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *JSONDecode) Type() value.Type { return value.JSON }

func (this *JSONDecode) Evaluate(item value.Value, context Context) (value.Value, error) {
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
func (this *JSONDecode) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.Actual().(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return value.NULL_VALUE, nil
	}

	var p interface{}
	err := json.Unmarshal([]byte(s), &p)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(p), nil
}

/*
Factory method pattern.
*/
func (this *JSONDecode) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewJSONDecode(operands[0])
	}
}

///////////////////////////////////////////////////
//
// JSONEncode
//
///////////////////////////////////////////////////

/*
This represents the  json function JSON_ENCODE(expr).
It marshals the N1QL value into a JSON-encoded string.
A MISSING becomes the empty string.
*/
type JSONEncode struct {
	UnaryFunctionBase
}

func NewJSONEncode(operand Expression) Function {
	rv := &JSONEncode{
		*NewUnaryFunctionBase("json_encode", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *JSONEncode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *JSONEncode) Type() value.Type { return value.STRING }

func (this *JSONEncode) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns a Json encoded string by sing the MarshalJSON
method. The return bytes value is cast to a string and returned.
*/
func (this *JSONEncode) Apply(context Context, arg value.Value) (value.Value, error) {
	bytes, _ := arg.MarshalJSON()
	return value.NewValue(string(bytes)), nil
}

/*
Factory method pattern.
*/
func (this *JSONEncode) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewJSONEncode(operands[0])
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

func NewEncodedSize(operand Expression) Function {
	rv := &EncodedSize{
		*NewUnaryFunctionBase("encoded_size", operand),
	}

	rv.expr = rv
	return rv
}

/*
visitor pattern.
*/
func (this *EncodedSize) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *EncodedSize) Type() value.Type { return value.NUMBER }

func (this *EncodedSize) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *EncodedSize) Apply(context Context, arg value.Value) (value.Value, error) {
	bytes, _ := arg.MarshalJSON()
	return value.NewValue(float64(len(bytes))), nil
}

/*
Factory method pattern.
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
it returns a NULL.
*/
type PolyLength struct {
	UnaryFunctionBase
}

func NewPolyLength(operand Expression) Function {
	rv := &PolyLength{
		*NewUnaryFunctionBase("poly_length", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *PolyLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *PolyLength) Type() value.Type { return value.NUMBER }

func (this *PolyLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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
Factory method pattern.
*/
func (this *PolyLength) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPolyLength(operands[0])
	}
}
