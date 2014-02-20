//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"reflect"

	"github.com/couchbaselabs/query/value"
)

// Commutative and associative operators.
type binary interface {
	Expression
	evaluate(first, second value.Value) (value.Value, error)
	isReverse(other Expression) bool
}

type binaryBase struct {
	expressionBase
	first  Expression
	second Expression
}

func (this *binaryBase) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, e := this.first.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	second, e := this.second.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	return binary(this).evaluate(first, second)
}

func (this *binaryBase) EquivalentTo(other Expression) bool {
	if reflect.TypeOf(this) == reflect.TypeOf(other) {
		o := other.(*binaryBase)
		return this.first.EquivalentTo(o.first) &&
			this.second.EquivalentTo(o.second)
	}

	if this.isReverse(other) {
		o := other.(*binaryBase)
		return this.first.EquivalentTo(o.second) &&
			this.second.EquivalentTo(o.first)
	}

	return false
}

func (this *binaryBase) Dependencies() Expressions {
	return Expressions{this.first, this.second}
}

func (this *binaryBase) Fold() Expression {
	this.first = this.first.Fold()
	this.second = this.second.Fold()

	switch f := this.first.(type) {
	case *Constant:
		switch s := this.second.(type) {
		case *Constant:
			v, e := binary(this).evaluate(f.Value(), s.Value())
			if e != nil {
				return this
			}
			return NewConstant(v)
		}
	}

	return this
}

func (this *binaryBase) evaluate(first, second value.Value) (value.Value, error) {
	panic("Must override.")
}

func (this *binaryBase) isReverse(other Expression) bool {
	panic("Must override.")
}
