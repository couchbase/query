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
	"encoding/base64"

	"github.com/couchbaselabs/query/value"
)

///////////////////////////////////////////////////
//
// Base64
//
///////////////////////////////////////////////////

type Base64 struct {
	UnaryFunctionBase
}

func NewBase64(operand Expression) Function {
	return &Base64{
		*NewUnaryFunctionBase("base64", operand),
	}
}

func (this *Base64) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Base64) Type() value.Type { return value.STRING }

func (this *Base64) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Base64) Apply(context Context, operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return operand, nil
	}

	bytes, _ := operand.MarshalJSON()
	str := base64.StdEncoding.EncodeToString(bytes)
	return value.NewValue(str), nil
}

func (this *Base64) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBase64(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Meta
//
///////////////////////////////////////////////////

type Meta struct {
	UnaryFunctionBase
}

func NewMeta(operand Expression) Function {
	return &Meta{
		*NewUnaryFunctionBase("meta", operand),
	}
}

func (this *Meta) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Meta) Type() value.Type { return value.OBJECT }

func (this *Meta) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Meta) Apply(context Context, operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return operand, nil
	}

	switch operand := operand.(type) {
	case value.AnnotatedValue:
		return value.NewValue(operand.GetAttachment("meta")), nil
	default:
		return value.NULL_VALUE, nil
	}
}

func (this *Meta) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMeta(operands[0])
	}
}
