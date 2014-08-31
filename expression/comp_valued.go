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

type IsValued struct {
	UnaryFunctionBase
}

func NewIsValued(operand Expression) Function {
	return &IsValued{
		*NewUnaryFunctionBase("isvalued", operand),
	}
}

func (this *IsValued) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsValued(this)
}

func (this *IsValued) Type() value.Type { return value.BOOLEAN }

func (this *IsValued) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsValued) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NewValue(arg.Type() > value.NULL), nil
}

func (this *IsValued) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsValued(operands[0])
	}
}

func NewIsNotValued(operand Expression) Expression {
	return NewNot(NewIsValued(operand))
}
