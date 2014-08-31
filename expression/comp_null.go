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

type IsNull struct {
	UnaryFunctionBase
}

func NewIsNull(operand Expression) Function {
	return &IsNull{
		*NewUnaryFunctionBase("isnull", operand),
	}
}

func (this *IsNull) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIsNull(this)
}

func (this *IsNull) Type() value.Type { return value.BOOLEAN }

func (this *IsNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *IsNull) Apply(context Context, arg value.Value) (value.Value, error) {
	switch arg.Type() {
	case value.NULL:
		return value.TRUE_VALUE, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.FALSE_VALUE, nil
	}
}

func (this *IsNull) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsNull(operands[0])
	}
}

func NewIsNotNull(operand Expression) Expression {
	return NewNot(NewIsNull(operand))
}
