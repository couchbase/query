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
	"strings"

	"github.com/couchbaselabs/query/value"
)

type Field struct {
	BinaryFunctionBase
	caseInsensitive bool
}

func NewField(first, second Expression) *Field {
	rv := &Field{
		BinaryFunctionBase: *NewBinaryFunctionBase("field", first, second),
	}

	rv.expr = rv
	return rv
}

func (this *Field) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitField(this)
}

func (this *Field) Type() value.Type { return value.JSON }

func (this *Field) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

func (this *Field) Alias() string {
	return this.Second().Alias()
}

func (this *Field) Apply(context Context, first, second value.Value) (value.Value, error) {
	switch second.Type() {
	case value.STRING:
		s := second.Actual().(string)
		v, ok := first.Field(s)

		if !ok && this.caseInsensitive {
			s = strings.ToLower(s)
			fields := first.Fields()
			for f, val := range fields {
				if s == strings.ToLower(f) {
					return value.NewValue(val), nil
				}
			}
		}

		return v, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		if first.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else {
			return value.NULL_VALUE, nil
		}
	}
}

func (this *Field) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewField(operands[0], operands[1])
	}
}

func (this *Field) Set(item, val value.Value, context Context) bool {
	first, er := this.First().Evaluate(item, context)
	if er != nil {
		return false
	}

	second, er := this.Second().Evaluate(item, context)
	if er != nil {
		return false
	}

	target := first
	if first.Type() == value.MISSING {
		target = item
	}
	switch second.Type() {
	case value.STRING:
		er := target.SetField(second.Actual().(string), val)
		return er == nil
	default:
		return false
	}
}

func (this *Field) Unset(item value.Value, context Context) bool {
	second, er := this.Second().Evaluate(item, context)
	if er != nil {
		return false
	}

	first, er := this.First().Evaluate(item, context)
	if er != nil {
		return false
	}

	target := first
	if first.Type() == value.MISSING {
		target = item
	}
	switch second.Type() {
	case value.STRING:
		er := target.UnsetField(second.Actual().(string))
		return er == nil
	default:
		return false
	}
}

func (this *Field) CaseInsensitive() bool {
	return this.caseInsensitive
}

func (this *Field) SetCaseInsensitive(insensitive bool) {
	this.caseInsensitive = insensitive
}

type FieldName struct {
	Constant
	name string
}

func NewFieldName(name string) Expression {
	rv := &FieldName{
		Constant: Constant{
			value: value.NewValue(name),
		},
		name: name,
	}

	rv.expr = rv
	return rv
}

func (this *FieldName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFieldName(this)
}

func (this *FieldName) Alias() string {
	return this.name
}
