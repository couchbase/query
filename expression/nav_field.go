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

type Field struct {
	binaryBase
}

func NewField(first, second Expression) Path {
	return &Field{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *Field) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Field) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Field) Alias() string {
	return this.second.Alias()
}

func (this *Field) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Field) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, allowed, keyspace)
}

func (this *Field) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Field) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Field) eval(first, second value.Value) (value.Value, error) {
	switch second.Type() {
	case value.STRING:
		s := second.Actual().(string)
		v, _ := first.Field(s)
		return v, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	}

	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

func (this *Field) Set(item, val value.Value, context Context) bool {
	second, er := this.second.Evaluate(item, context)
	if er != nil {
		return false
	}

	switch second.Type() {
	case value.STRING:
		er := item.SetField(second.Actual().(string), val)
		return er == nil
	default:
		return false
	}
}

func (this *Field) Unset(item value.Value, context Context) bool {
	second, er := this.second.Evaluate(item, context)
	if er != nil {
		return false
	}

	switch second.Type() {
	case value.STRING:
		er := item.UnsetField(second.Actual().(string))
		return er == nil
	default:
		return false
	}
}

type FieldName struct {
	Constant
	name string
}

func NewFieldName(name string) Expression {
	return &FieldName{
		Constant: Constant{
			value: value.NewValue(name),
		},
		name: name,
	}
}

func (this *FieldName) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *FieldName:
		return this.value.Equals(other.value)
	default:
		return false
	}
}

func (this *FieldName) Fold() (Expression, error) {
	return this, nil
}

func (this *FieldName) Formalize(allowed value.Value,
	keyspace string) (Expression, error) {
	return this, nil
}

func (this *FieldName) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *FieldName) VisitChildren(visitor Visitor) (Expression, error) {
	return this, nil
}

func (this *FieldName) Alias() string {
	return this.name
}
