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
Represents IMod for arithmetic expressions. Type IMod is a struct
that implements BinaryFunctionBase.
*/
type IMod struct {
	BinaryFunctionBase
}

func NewIMod(first, second Expression) Function {
	rv := &IMod{
		*NewBinaryFunctionBase("imod", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *IMod) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IMod) Type() value.Type { return value.NUMBER }

func (this *IMod) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *IMod) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	if first.Type() == value.NUMBER && second.Type() == value.NUMBER {
		return first.(value.NumberValue).IMod(second.(value.NumberValue)), nil
	} else {
		return value.NULL_VALUE, nil
	}
}

/*
Factory method pattern.
*/
func (this *IMod) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIMod(operands[0], operands[1])
	}
}
