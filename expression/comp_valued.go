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

type IsValued struct {
	unaryBase
}

func NewIsValued(operand Expression) Expression {
	return &IsValued{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *IsValued) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IsValued) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IsValued) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IsValued) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IsValued) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IsValued) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IsValued) eval(operand value.Value) (value.Value, error) {
	if operand.Type() > value.NULL {
		return value.NewValue(true), nil
	} else {
		return value.NewValue(false), nil
	}
}

func NewIsNotValued(operand Expression) Expression {
	return NewNot(NewIsValued(operand))
}
