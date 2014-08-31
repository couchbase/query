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

type Within struct {
	BinaryFunctionBase
}

func NewWithin(first, second Expression) Function {
	return &Within{
		*NewBinaryFunctionBase("within", first, second),
	}
}

func (this *Within) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWithin(this)
}

func (this *Within) Type() value.Type { return value.BOOLEAN }

func (this *Within) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *Within) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.ARRAY && second.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	desc := second.Descendants(make([]interface{}, 0, 64))
	for _, d := range desc {
		if first.Equals(value.NewValue(d)) {
			return value.TRUE_VALUE, nil
		}
	}

	return value.FALSE_VALUE, nil
}

func (this *Within) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewWithin(operands[0], operands[1])
	}
}

func NewNotWithin(first, second Expression) Expression {
	return NewNot(NewWithin(first, second))
}
