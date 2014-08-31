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

type Between struct {
	TernaryFunctionBase
}

func NewBetween(item, low, high Expression) Function {
	return &Between{
		*NewTernaryFunctionBase("between", item, low, high),
	}
}

func (this *Between) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBetween(this)
}

func (this *Between) Type() value.Type { return value.BOOLEAN }

func (this *Between) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *Between) Apply(context Context, item, low, high value.Value) (value.Value, error) {
	if item.Type() == value.MISSING || low.Type() == value.MISSING || high.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if item.Type() == value.NULL || low.Type() == value.NULL || high.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(item.Collate(low) >= 0 && item.Collate(high) <= 0), nil
}

func (this *Between) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBetween(operands[0], operands[1], operands[2])
	}
}

func NewNotBetween(item, low, high Expression) Expression {
	return NewNot(NewBetween(item, low, high))
}
