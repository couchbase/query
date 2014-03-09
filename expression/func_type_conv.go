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

type ToArray struct {
	unaryBase
}

func NewToArray(arg Expression) Function {
	return &ToArray{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ToArray) evaluate(arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	} else if arg.Type() == value.ARRAY {
		return arg, nil
	}

	return value.NewValue([]interface{}{arg}), nil
}

func (this *ToArray) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewToArray(args[0])
	}
}

type ToAtom struct {
	unaryBase
}

func NewToAtom(arg Expression) Function {
	return &ToAtom{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ToAtom) evaluate(arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.BOOLEAN, value.NUMBER, value.STRING, value.MISSING, value.NULL:
		return arg, nil
	default:
		switch a := arg.Actual().(type) {
		case []interface{}:
			if len(a) == 1 {
				return value.NewValue(a[0]), nil
			}
		case map[string]interface{}:
			if len(a) == 1 {
				for _, v := range a {
					return value.NewValue(v), nil
				}
			}
		}
	}

	return value.NULL_VALUE, nil
}

func (this *ToAtom) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewToAtom(args[0])
	}
}

type ToBool struct {
	unaryBase
}

func NewToBool(arg Expression) Function {
	return &ToBool{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ToBool) evaluate(arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.MISSING, value.NULL, value.BOOLEAN:
		return arg, nil
	default:
		switch a := arg.Actual().(type) {
		case float64:
			if a == 0 || math.IsNaN(a) {
				return value.NewValue(false), nil
			}
		case string:
			if len(a) == 0 {
				return value.NewValue(false), nil
			}
		case []interface{}:
			if len(a) == 0 {
				return value.NewValue(false), nil
			}
		case map[string]interface{}:
			if len(a) == 0 {
				return value.NewValue(false), nil
			}
		}
	}

	return value.NULL_VALUE, nil
}

func (this *ToBool) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewToBool(args[0])
	}
}

type ToNum struct {
	unaryBase
}

func NewToNum(arg Expression) Function {
	return &ToNum{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ToNum) evaluate(arg value.Value) (value.Value, error) {
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
			f, e := strconv.ParseFloat(a, 64)
			if e == nil {
				return value.NewValue(f), nil
			}
		}
	}

	return value.NULL_VALUE, nil
}

func (this *ToNum) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewToNum(args[0])
	}
}

type ToStr struct {
	unaryBase
}

func NewToStr(arg Expression) Function {
	return &ToStr{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ToStr) evaluate(arg value.Value) (value.Value, error) {
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
	return func(args Expressions) Function {
		return NewToStr(args[0])
	}
}
