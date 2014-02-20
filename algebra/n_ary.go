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
	evaluate(operands value.Values) (value.Value, error)
	construct(constant value.Value, others Expressions) Expression
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
	operands := make(Expressions, 0, len(this.operands))
	for _, o := range this.operands {
		o = o.Fold()
		if reflect.TypeOf(this) == reflect.TypeOf(o) {
			// Associative, so promote subexpressions.
			for _, oo := range o.(*nAryBase).operands {
				operands = append(operands, oo)
			}
		} else {
			operands = append(operands, o)
		}
	}

	this.operands = operands

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

		if len(others) == 0 {
			return NewConstant(c)
		}

		return nary.construct(c, others)
	}

	return this
}

func (this *nAryBase) evaluate(operands value.Values) (value.Value, error) {
	panic("Must override.")
}

func (this *nAryBase) construct(constant value.Value, others Expressions) Expression {
	panic("Must override.")
}

var _MISSING_VALUE = value.NewMissingValue()
var _NULL_VALUE = value.NewValue(nil)
