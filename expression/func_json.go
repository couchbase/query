//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"encoding/json"
	"strings"

	"github.com/couchbase/query/util"
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

/*
This method returns a valid N1QL value from a JSON encoded
string. If the input type is missing return missing, and if
it isnt string then return null value. Conver the input arg
to valid Go type and cast to a string. If it is an empty
string return missing value. If not then call the Unmarshal
method defined in the json package, by casting the strings
to a bytes slice and return the json value.
*/
func (this *JSONDecode) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := arg.ToString()
	s = strings.TrimSpace(s)
	if s == "" {
		return value.NULL_VALUE, nil
	}

	var p interface{}
	err = json.Unmarshal([]byte(s), &p)
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

/*
This method returns a Json encoded string by sing the MarshalJSON
method. The return bytes value is cast to a string and returned.
*/
func (this *JSONEncode) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	arg.Actual() // force unwrapping of parsed values
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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	arg.Actual() // force unwrapping of parsed values
	bytes, _ := arg.MarshalJSON()
	return value.NewValue(len(bytes)), nil
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
// Pairs
//
///////////////////////////////////////////////////

/*
Dynamic index for data platforms.
*/
type Pairs struct {
	UnaryFunctionBase
}

func NewPairs(operand Expression) Function {
	rv := &Pairs{
		*NewUnaryFunctionBase("pairs", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Pairs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Pairs) Type() value.Type { return value.ARRAY }

func (this *Pairs) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	var bufcap int

	actual := arg.Actual()
	switch arg.Type() {
	case value.OBJECT:
		bufcap = 2 * len(actual.(map[string]interface{}))
	case value.ARRAY:
		bufcap = 2 * len(actual.([]interface{}))
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.NULL_VALUE, nil
	}

	buffer := traversePairs(actual, make([]interface{}, 0, bufcap))
	return value.NewValue(buffer), nil
}

/*
Factory method pattern.
*/
func (this *Pairs) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPairs(operands[0])
	}
}

///////////////////////////////////////////////////
//
// PolyLength
//
///////////////////////////////////////////////////

/*
This represents the json function POLY_LENGTH(expr). It returns the
length of the value after evaluating the expression. The exact meaning
of length depends on the type of the value. For a string it returns
the length of the string, for array it returns the number of elements,
for objects it returns the number of name/value pairs in the object
and for any other value it returns NULL or MISSING.
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
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	switch oa := arg.Actual().(type) {
	case string:
		return value.NewValue(len(oa)), nil
	case []interface{}:
		return value.NewValue(len(oa)), nil
	case map[string]interface{}:
		return value.NewValue(len(oa)), nil
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

func traversePairs(actual interface{}, buffer []interface{}) []interface{} {
	length := 0

	switch actual := actual.(type) {
	case map[string]interface{}:
		length = len(actual)
	case []interface{}:
		length = len(actual)
	default:
		// Do nothing
	}

	if length == 0 {
		return buffer
	}

	buffer = ensureBuffer(buffer, len(buffer)+length)

	switch actual := actual.(type) {
	case map[string]interface{}:
		for n, v := range actual {
			buffer = append(buffer, []interface{}{n, v})

			v = value.NewValue(v).Actual()

			switch v := v.(type) {
			case []interface{}:
				buffer = ensureBuffer(buffer, len(buffer)+len(v))
				for _, vv := range v {
					buffer = append(buffer, []interface{}{n, vv})
				}
			}

			buffer = traversePairs(v, buffer)
		}
	case []interface{}:
		for _, v := range actual {
			v = value.NewValue(v).Actual()
			buffer = traversePairs(v, buffer)
		}
	}

	return buffer
}

func ensureBuffer(buffer []interface{}, length int) []interface{} {
	if cap(buffer) >= length {
		return buffer
	}

	buffer2 := make([]interface{}, 0, util.MaxInt(length, 2*cap(buffer)))
	return append(buffer2, buffer...)
}

var _SET_POOL = value.NewSetPool(64, true, false)
