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

type IsArray struct {
	UnaryFunctionBase
}

func NewIsArray(operand Expression) Function {
	return &IsArray{
		*NewUnaryFunctionBase("isarray", operand),
	}
}

func (this *IsArray) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsArray) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsArray) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.ARRAY), nil
}

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

type IsAtom struct {
	UnaryFunctionBase
}

func NewIsAtom(operand Expression) Function {
	return &IsAtom{
		*NewUnaryFunctionBase("isatom", operand),
	}
}

func (this *IsAtom) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsAtom) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsAtom) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.BOOLEAN, value.NUMBER, value.STRING:
		return value.NewValue(true), nil
	default:
		return value.NewValue(false), nil
	}
}

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

type IsBool struct {
	UnaryFunctionBase
}

func NewIsBool(operand Expression) Function {
	return &IsBool{
		*NewUnaryFunctionBase("isbool", operand),
	}
}

func (this *IsBool) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsBool) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsBool) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.BOOLEAN), nil
}

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

type IsNum struct {
	UnaryFunctionBase
}

func NewIsNum(operand Expression) Function {
	return &IsNum{
		*NewUnaryFunctionBase("isnum", operand),
	}
}

func (this *IsNum) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsNum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsNum) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.NUMBER), nil
}

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

type IsObj struct {
	UnaryFunctionBase
}

func NewIsObj(operand Expression) Function {
	return &IsObj{
		*NewUnaryFunctionBase("isobj", operand),
	}
}

func (this *IsObj) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsObj) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsObj) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.OBJECT), nil
}

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

type IsStr struct {
	UnaryFunctionBase
}

func NewIsStr(operand Expression) Function {
	return &IsStr{
		*NewUnaryFunctionBase("isstr", operand),
	}
}

func (this *IsStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsStr) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.STRING), nil
}

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

type TypeName struct {
	UnaryFunctionBase
}

func NewTypeName(operand Expression) Function {
	return &TypeName{
		*NewUnaryFunctionBase("typename", operand),
	}
}

func (this *TypeName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *TypeName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *TypeName) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type().String()), nil
}

func (this *TypeName) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewTypeName(operands[0])
	}
}
