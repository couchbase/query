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
	"github.com/couchbase/query/value"
)

/*
Represents integer div for arithmetic expressions. Type IDiv is a
struct that implements BinaryFunctionBase.
*/
type IDiv struct {
	BinaryFunctionBase
}

func NewIDiv(first, second Expression) Function {
	rv := &IDiv{
		*NewBinaryFunctionBase("idiv", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IDiv) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IDiv) Type() value.Type { return value.NUMBER }

func (this *IDiv) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *IDiv) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if first.Type() == value.NUMBER && second.Type() == value.NUMBER {
		return value.AsNumberValue(first).IDiv(value.AsNumberValue(second)), nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *IDiv) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIDiv(operands[0], operands[1])
	}
}
