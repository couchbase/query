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
	"math"

	"github.com/couchbaselabs/query/value"
)

type Element struct {
	BinaryFunctionBase
}

func NewElement(first, second Expression) *Element {
	return &Element{
		*NewBinaryFunctionBase("element", first, second),
	}
}

func (this *Element) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitElement(this)
}

func (this *Element) Type() value.Type { return value.JSON }

func (this *Element) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *Element) Apply(context Context, first, second value.Value) (value.Value, error) {
	switch second.Type() {
	case value.NUMBER:
		s := second.Actual().(float64)
		if s == math.Trunc(s) {
			v, _ := first.Index(int(s))
			return v, nil
		}
	case value.MISSING:
		return value.MISSING_VALUE, nil
	}

	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

func (this *Element) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewElement(operands[0], operands[1])
	}
}

func (this *Element) Set(item, val value.Value, context Context) bool {
	second, er := this.Second().Evaluate(item, context)
	if er != nil {
		return false
	}

	first, er := this.First().Evaluate(item, context)
	if er != nil {
		return false
	}

	switch second.Type() {
	case value.NUMBER:
		s := second.Actual().(float64)
		if s == math.Trunc(s) {
			er := first.SetIndex(int(s), val)
			return er == nil
		}
	}

	return false
}

func (this *Element) Unset(item value.Value, context Context) bool {
	return false
}
