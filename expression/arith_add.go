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

type Add struct {
	caNAryBase
}

func NewAdd(operands ...Expression) Expression {
	return &Add{
		caNAryBase{
			nAryBase{
				operands: operands,
			},
		},
	}
}

func (this *Add) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Add) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Add) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Add) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *Add) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Add) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Add) eval(operands value.Values) (value.Value, error) {
	null := false
	sum := 0.0
	for _, v := range operands {
		if !null && v.Type() == value.NUMBER {
			sum += v.Actual().(float64)
		} else if v.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else {
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(sum), nil
}

func (this *Add) construct(constant value.Value, others Expressions) Expression {
	if constant.Type() == value.MISSING {
		return NewConstant(constant)
	} else if constant.Type() == value.NUMBER && constant.Actual().(float64) == 0.0 {
		return NewAdd(others...)
	}

	return NewAdd(append(others, NewConstant(constant))...)
}
