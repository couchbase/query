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

type DecodeJSON struct {
	UnaryFunctionBase
}

func NewDecodeJSON(operand Expression) Function {
	return &DecodeJSON{
		*NewUnaryFunctionBase("decode_json", operand),
	}
}

func (this *DecodeJSON) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DecodeJSON) Type() value.Type { return value.JSON }

func (this *DecodeJSON) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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

type EncodeJSON struct {
	UnaryFunctionBase
}

func NewEncodeJSON(operand Expression) Function {
	return &EncodeJSON{
		*NewUnaryFunctionBase("encode_json", operand),
	}
}

func (this *EncodeJSON) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *EncodeJSON) Type() value.Type { return value.STRING }

func (this *EncodeJSON) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *EncodeJSON) Apply(context Context, arg value.Value) (value.Value, error) {
	bytes, _ := arg.MarshalJSON()
	return value.NewValue(string(bytes)), nil
}

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

type EncodedSize struct {
	UnaryFunctionBase
}

func NewEncodedSize(operand Expression) Function {
	return &EncodedSize{
		*NewUnaryFunctionBase("encoded_size", operand),
	}
}

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

type PolyLength struct {
	UnaryFunctionBase
}

func NewPolyLength(operand Expression) Function {
	return &PolyLength{
		*NewUnaryFunctionBase("poly_length", operand),
	}
}

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

func (this *PolyLength) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewPolyLength(operands[0])
	}
}
