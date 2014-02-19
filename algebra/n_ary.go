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
type nAry interface {
	Expression
	constructor() nAryConstructor
	evaluate(operands value.Values) (value.Value, error)
	shortCircuit() bool
}

type nAryConstructor func(operands Expressions) Expression

type nAryBase struct {
	expressionBase
	operands Expressions
}

func (this *nAryBase) Evaluate(item value.Value, context Context) (value.Value, error) {
	var e error
	operands := make([]value.Value, len(this.operands))
	for i, o := range this.operands {
		operands[i], e = o.Evaluate(item, context)
		if e != nil {
			return nil, e
		}
	}

	return nAry(this).evaluate(operands)
}

func (this *nAryBase) EquivalentTo(other Expression) bool {
	if reflect.TypeOf(this) != reflect.TypeOf(other) {
		return false
	}

	that := other.(*nAryBase)
	if len(this.operands) != len(that.operands) {
		return false
	}

	found := make([]bool, len(this.operands))

	for _, first := range this.operands {
		for j, second := range that.operands {
			if !found[j] && first.EquivalentTo(second) {
				found[j] = true
				break
			}
		}
	}

	for _, f := range found {
		if !f {
			return false
		}
	}

	return true
}

func (this *nAryBase) Dependencies() Expressions {
	return this.operands
}

func (this *nAryBase) Fold() Expression {
	operands := this.operands
	for i, o := range operands {
		operands[i] = o.Fold()
	}

	constants := make(value.Values, 0, len(operands))
	others := make(Expressions, 0, len(operands))
	for i, o := range operands {
		switch o := o.(type) {
		case *Constant:
			constants[i] = o.Value()
		default:
			others[i] = o
		}
	}

	if len(constants) > 0 {
		nary := nAry(this)
		c, e := nary.evaluate(constants)
		if e != nil {
			return this
		}

		constant := NewConstant(c)
		if len(others) == 0 || nary.shortCircuit() {
			return constant
		}

		others = append(others, constant)
		return nary.constructor()(others)
	}

	return this
}

func (this *nAryBase) constructor() nAryConstructor {
	panic("Must override.")
}

func (this *nAryBase) evaluate(operands value.Values) (value.Value, error) {
	panic("Must override.")
}

func (this *nAryBase) shortCircuit() bool {
	return false
}

var _MISSING_VALUE = value.NewMissingValue()
var _NULL_VALUE = value.NewValue(nil)

func unvalued(operands ...value.Value) value.Value {
	if len(operands) == 0 {
		return _MISSING_VALUE
	}

	null := false
	for _, v := range operands {
		if v.Type() == value.MISSING {
			return v
		}

		if v.Type() == value.NULL {
			null = true
		}
	}

	if null {
		return _NULL_VALUE
	}

	return nil
}
