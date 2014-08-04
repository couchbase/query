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

type Exists struct {
	unaryBase
}

func NewExists(operand Expression) *Exists {
	return &Exists{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *Exists) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *Exists) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *Exists) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *Exists) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *Exists) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Exists) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *Exists) eval(operand value.Value) (value.Value, error) {
	if operand.Type() == value.ARRAY {
		a := operand.Actual().([]interface{})
		return value.NewValue(len(a) > 0), nil
	} else if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else {
		return value.NULL_VALUE, nil
	}
}

func (this *Exists) Operand() Expression {
	return this.operand
}
