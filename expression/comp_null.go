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

type IsNull struct {
	unaryBase
}

func NewIsNull(operand Expression) Expression {
	return &IsNull{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *IsNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IsNull) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IsNull) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IsNull) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IsNull) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IsNull) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IsNull) eval(operand value.Value) (value.Value, error) {
	switch operand.Type() {
	case value.NULL:
		return value.NewValue(true), nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.NewValue(false), nil
	}
}

func NewIsNotNull(operand Expression) Expression {
	return NewNot(NewIsNull(operand))
}
