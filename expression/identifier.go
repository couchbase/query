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
	"fmt"

	"github.com/couchbaselabs/query/value"
)

type Identifier struct {
	ExpressionBase
	identifier string
}

func NewIdentifier(identifier string) Path {
	return &Identifier{
		identifier: identifier,
	}
}

func (this *Identifier) Evaluate(item value.Value, context Context) (value.Value, error) {
	rv, _ := item.Field(this.identifier)
	return rv, nil
}

func (this *Identifier) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *Identifier:
		return this.identifier == other.identifier
	default:
		return false
	}
}

func (this *Identifier) Alias() string {
	return this.identifier
}

func (this *Identifier) Fold() (Expression, error) {
	return this, nil
}

// Formal notation; qualify fields with keyspace name.
// Identifiers in "allowed" are left unmodified.
// Any other identifier is qualified with keyspace; if keyspace is empty, then error.
func (this *Identifier) Formalize(allowed value.Value, keyspace string) (Expression, error) {
	_, ok := allowed.Field(this.identifier)
	if ok {
		return this, nil
	}

	if keyspace == "" {
		return nil, fmt.Errorf("Ambiguous reference to field %v.", this.identifier)
	}

	return NewField(NewIdentifier(keyspace), NewFieldName(this.identifier)), nil
}

func (this *Identifier) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *Identifier) VisitChildren(visitor Visitor) (Expression, error) {
	return this, nil
}

func (this *Identifier) Set(item, val value.Value, context Context) bool {
	er := item.SetField(this.identifier, val)
	return er == nil
}

func (this *Identifier) Unset(item value.Value, context Context) bool {
	er := item.UnsetField(this.identifier)
	return er == nil
}
