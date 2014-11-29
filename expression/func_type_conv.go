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

type ToArray struct {
	UnaryFunctionBase
}

func NewToArray(operand Expression) Function {
	rv := &ToArray{
		*NewUnaryFunctionBase("to_array", operand),
	}

	rv.expr = rv
	return rv
}

func (this *ToArray) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToArray) Type() value.Type { return value.ARRAY }

func (this *ToArray) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ToArray) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	} else if arg.Type() == value.ARRAY {
		return arg, nil
	}

	return value.NewValue([]interface{}{arg.Actual()}), nil
}

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

type ToAtom struct {
	UnaryFunctionBase
}

func NewToAtom(operand Expression) Function {
	rv := &ToAtom{
		*NewUnaryFunctionBase("to_atom", operand),
	}

	rv.expr = rv
	return rv
}

func (this *ToAtom) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToAtom) Type() value.Type {
	t := this.Operand().Type()
	if t < value.ARRAY {
		return t
	} else {
		return value.JSON
	}
}

func (this *ToAtom) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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

type ToBool struct {
	UnaryFunctionBase
}

func NewToBool(operand Expression) Function {
	rv := &ToBool{
		*NewUnaryFunctionBase("to_bool", operand),
	}

	rv.expr = rv
	return rv
}

func (this *ToBool) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToBool) Type() value.Type { return value.BOOLEAN }

func (this *ToBool) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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

type ToNum struct {
	UnaryFunctionBase
}

func NewToNum(operand Expression) Function {
	rv := &ToNum{
		*NewUnaryFunctionBase("to_num", operand),
	}

	rv.expr = rv
	return rv
}

func (this *ToNum) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToNum) Type() value.Type { return value.NUMBER }

func (this *ToNum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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

type ToObj struct {
	UnaryFunctionBase
}

func NewToObj(operand Expression) Function {
	rv := &ToObj{
		*NewUnaryFunctionBase("to_obj", operand),
	}

	rv.expr = rv
	return rv
}

func (this *ToObj) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToObj) Type() value.Type { return value.OBJECT }

func (this *ToObj) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

var _EMPTY_OBJECT = value.NewValue(map[string]interface{}{})

func (this *ToObj) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING, value.NULL, value.OBJECT:
		return arg, nil
	}

	return _EMPTY_OBJECT, nil
}

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

type ToStr struct {
	UnaryFunctionBase
}

func NewToStr(operand Expression) Function {
	rv := &ToStr{
		*NewUnaryFunctionBase("to_str", operand),
	}

	rv.expr = rv
	return rv
}

func (this *ToStr) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ToStr) Type() value.Type { return value.STRING }

func (this *ToStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

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

func (this *ToStr) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewToStr(operands[0])
	}
}
