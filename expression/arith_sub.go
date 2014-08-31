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

type Sub struct {
	BinaryFunctionBase
}

func NewSub(first, second Expression) Function {
	return &Sub{
		*NewBinaryFunctionBase("sub", first, second),
	}
}

func (this *Sub) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSub(this)
}

func (this *Sub) Type() value.Type { return value.NUMBER }

func (this *Sub) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *Sub) Apply(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.NUMBER && second.Type() == value.NUMBER {
		diff := first.Actual().(float64) - second.Actual().(float64)
		return value.NewValue(diff), nil
	} else if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

func (this *Sub) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewSub(operands[0], operands[1])
	}
}
