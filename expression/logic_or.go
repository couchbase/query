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

type Or struct {
	caNAryBase
}

func NewOr(operands ...Expression) Expression {
	return &Or{
		caNAryBase{
			nAryBase{
				operands: operands,
			},
		},
	}
}

func (this *Or) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Or) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Or) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Or) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *Or) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Or) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Or) eval(operands value.Values) (value.Value, error) {
	missing := false
	null := false
	for _, v := range operands {
		if v.Type() > value.NULL {
			if v.Truth() {
				return value.NewValue(true), nil
			}
		} else if v.Type() == value.NULL {
			null = true
		} else if v.Type() == value.MISSING {
			missing = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	} else if missing {
		return value.MISSING_VALUE, nil
	} else {
		return value.NewValue(false), nil
	}
}

func (this *Or) construct(constant value.Value, others Expressions) Expression {
	if constant.Truth() {
		return NewConstant(value.NewValue(true))
	}

	return NewOr(append(others, NewConstant(constant))...)
}
