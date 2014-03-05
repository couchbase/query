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
	json "github.com/dustin/gojson"
)

type DecodeJSON struct {
	unaryBase
}

func NewDecodeJSON(operand Expression) Function {
	return &DecodeJSON{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *DecodeJSON) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	s := operand.Actual().(string)
	var p interface{}
	err := json.Unmarshal([]byte(s), &p)
	if err != nil {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(p), nil
}

func (this *DecodeJSON) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewDecodeJSON(args[0])
	}
}

type EncodeJSON struct {
	unaryBase
}

func NewEncodeJSON(operand Expression) Function {
	return &EncodeJSON{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *EncodeJSON) evaluate(operand value.Value) (value.Value, error) {
	return value.NewValue(string(operand.Bytes())), nil
}

func (this *EncodeJSON) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewEncodeJSON(args[0])
	}
}

type EncodedSize struct {
	unaryBase
}

func NewEncodedSize(operand Expression) Function {
	return &EncodedSize{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *EncodedSize) evaluate(operand value.Value) (value.Value, error) {
	return value.NewValue(float64(len(operand.Bytes()))), nil
}

func (this *EncodedSize) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewEncodedSize(args[0])
	}
}

type PolyLength struct {
	unaryBase
}

func NewPolyLength(operand Expression) Function {
	return &PolyLength{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *PolyLength) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	switch oa := operand.Actual().(type) {
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
	return func(args Expressions) Function {
		return NewPolyLength(args[0])
	}
}
