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

type EQ struct {
	binaryBase
}

func NewEQ(first, second Expression) Expression {
	return &EQ{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *EQ) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *EQ) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *EQ) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *EQ) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *EQ) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *EQ) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *EQ) eval(first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() == value.NULL || second.Type() == value.NULL ||
		first.Type() != second.Type() {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(first.Collate(second) == 0), nil
}

func NewNE(first, second Expression) Expression {
	return NewNot(NewEQ(first, second))
}
