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

type ArrayLiteral struct {
	nAryBase
}

func NewArrayLiteral(exprs Expressions) Expression {
	return &ArrayLiteral{
		nAryBase{
			operands: exprs,
		},
	}
}

func (this *ArrayLiteral) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ArrayLiteral) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ArrayLiteral) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ArrayLiteral) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ArrayLiteral) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ArrayLiteral) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ArrayLiteral) eval(operands value.Values) (value.Value, error) {
	a := make([]interface{}, len(operands))
	for i, o := range operands {
		a[i] = o
	}

	return value.NewValue(a), nil
}
