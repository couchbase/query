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

type IsMissing struct {
	UnaryFunctionBase
}

func NewIsMissing(operand Expression) Function {
	return &IsMissing{
		*NewUnaryFunctionBase("ismissing", operand),
	}
}

func (this *IsMissing) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsMissing(this)
}

func (this *IsMissing) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsMissing) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() == value.MISSING), nil
}

func (this *IsMissing) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsMissing(operands[0])
	}
}

func NewIsNotMissing(operand Expression) Expression {
	return NewNot(NewIsMissing(operand))
}
