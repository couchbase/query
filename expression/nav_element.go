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

	"github.com/couchbase/query/value"
)

/*
Nested expressions are used to access elements inside of arrays.
They support using the bracket notation ([position]) to access
elements inside an array.
*/
type Element struct {
	BinaryFunctionBase
}

func NewElement(first, second Expression) *Element {
	rv := &Element{
		*NewBinaryFunctionBase("element", first, second),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Element) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitElement(this)
}

func (this *Element) Type() value.Type { return value.JSON }

func (this *Element) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
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

/*
Factory method pattern.
*/
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

/*
Return false.
*/
func (this *Element) Unset(item value.Value, context Context) bool {
	return false
}
