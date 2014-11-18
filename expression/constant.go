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

type Constant struct {
	ExpressionBase
	value value.Value
}

var NULL_EXPR = NewConstant(value.NULL_VALUE)
var MISSING_EXPR = NewConstant(value.MISSING_VALUE)
var FALSE_EXPR = NewConstant(value.FALSE_VALUE)
var TRUE_EXPR = NewConstant(value.TRUE_VALUE)
var ZERO_EXPR = NewConstant(value.ZERO_VALUE)
var ONE_EXPR = NewConstant(value.ONE_VALUE)

func NewConstant(val interface{}) Expression {
	return &Constant{
		value: value.NewValue(val),
	}
}

func (this *Constant) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitConstant(this)
}

func (this *Constant) Type() value.Type { return this.value.Type() }

func (this *Constant) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.value, nil
}

func (this *Constant) Indexable() bool {
	return true
}

func (this *Constant) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *Constant:
		return this.value.Equals(other.value)
	default:
		return false
	}
}

func (this *Constant) SubsetOf(other Expression) bool {
	return this.EquivalentTo(other)
}

func (this *Constant) Children() Expressions {
	return nil
}

func (this *Constant) MapChildren(mapper Mapper) error {
	return nil
}

func (this *Constant) Value() value.Value {
	return this.value
}
