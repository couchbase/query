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

type LE struct {
	BinaryFunctionBase
}

func NewLE(first, second Expression) Function {
	return &LE{
		*NewBinaryFunctionBase("le", first, second),
	}
}

func (this *LE) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLE(this)
}

func (this *LE) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *LE) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() == value.NULL || second.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(first.Collate(second) <= 0), nil
}

func (this *LE) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewLE(operands[0], operands[1])
	}
}
