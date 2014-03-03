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
	"strings"

	"github.com/couchbaselabs/query/value"
)

type Contains struct {
	binaryBase
}

func NewContains(first, second Expression) Function {
	return &Contains{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Contains) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Contains(first.Actual().(string), second.Actual().(string))
	return value.NewValue(rv), nil
}

func (this *Contains) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewContains(args[0], args[1])
	}
}

type Length struct {
	unaryBase
}

func NewLength(operand Expression) Function {
	return &Length{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *Length) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := len(operand.Actual().(string))
	return value.NewValue(float64(rv)), nil
}

func (this *Length) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewLength(args[0])
	}
}

type Lower struct {
	unaryBase
}

func NewLower(operand Expression) Function {
	return &Lower{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *Lower) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.ToLower(operand.Actual().(string))
	return value.NewValue(rv), nil
}

func (this *Lower) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewLower(args[0])
	}
}

type LTrim struct {
	nAryBase
}

func NewLTrim(operands Expressions) Function {
	return &LTrim{
		nAryBase{
			operands: operands,
		},
	}
}

func (this *LTrim) MinArgs() int { return 1 }

func (this *LTrim) MaxArgs() int { return 2 }

func (this *LTrim) evaluate(operands value.Values) (value.Value, error) {
	null := false
	for _, o := range operands {
		if o.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if o.Type() != value.STRING {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	chars := _WHITESPACE
	if len(operands) > 1 {
		chars = operands[1]
	}

	rv := strings.TrimLeft(operands[0].Actual().(string), chars.Actual().(string))
	return value.NewValue(rv), nil
}

func (this *LTrim) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewLTrim(args)
	}
}

var _WHITESPACE = value.NewValue(" \t\n\f\r")

type Position struct {
	binaryBase
}

func NewPosition(first, second Expression) Function {
	return &Position{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Position) evaluate(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	rv := strings.Index(first.Actual().(string), second.Actual().(string))
	return value.NewValue(float64(rv)), nil
}

func (this *Position) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewPosition(args[0], args[1])
	}
}
