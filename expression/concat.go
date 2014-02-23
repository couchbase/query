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
	"bytes"

	"github.com/couchbaselabs/query/value"
)

type Concat struct {
	expressionBase
	operands Expressions
}

func NewConcat(operands ...Expression) Expression {
	return &Concat{
		operands: operands,
	}
}

func (this *Concat) Evaluate(item value.Value, context Context) (value.Value, error) {
	var e error
	operands := make([]value.Value, len(this.operands))
	for i, o := range this.operands {
		operands[i], e = o.Evaluate(item, context)
		if e != nil {
			return nil, e
		}
	}

	return this.evaluate(operands)
}

func (this *Concat) EquivalentTo(other Expression) bool {
	that, ok := other.(*Concat)
	if !ok {
		return false
	}

	if len(this.operands) != len(that.operands) {
		return false
	}

	for i, o := range this.operands {
		if !o.EquivalentTo(that.operands[i]) {
			return false
		}
	}

	return true
}

func (this *Concat) Dependencies() Expressions {
	return this.operands
}

func (this *Concat) Fold() Expression {
	operands := make(Expressions, 0, len(this.operands))
	for _, o := range this.operands {
		o = o.Fold()
		switch o := o.(type) {
		case *Concat:
			// Associative, so promote subexpressions.
			for _, oo := range o.operands {
				operands = append(operands, oo)
			}
		default:
			operands = append(operands, o)
		}
	}

	this.operands = operands

	constants := make(value.Values, 0, len(operands))
	others := make(Expressions, 0, len(operands))
	for _, o := range operands {
		switch o := o.(type) {
		case *Constant:
			constants = append(constants, o.Value())
		default:
			if len(constants) > 0 {
				c, e := this.evaluate(constants)
				if e != nil {
					return this
				}
				others = append(others, NewConstant(c))
				constants = make(value.Values, 0, len(operands))
			}

			others = append(others, o)
		}
	}

	if len(constants) > 0 {
		c, e := this.evaluate(constants)
		if e != nil {
			return this
		}

		if len(others) == 0 {
			return NewConstant(c)
		}

		others = append(others, NewConstant(c))
	}

	return NewConcat(others...)
}

func (this *Concat) evaluate(operands value.Values) (value.Value, error) {
	var buf bytes.Buffer
	null := false

	for _, o := range operands {
		switch o.Type() {
		case value.STRING:
			buf.WriteString(o.Actual().(string))
		case value.MISSING:
			return _MISSING_VALUE, nil
		default:
			null = true
		}
	}

	if null {
		return _NULL_VALUE, nil
	}

	return value.NewValue(buf.String()), nil
}
