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

type Identifier struct {
	ExpressionBase
	identifier string
}

func NewIdentifier(identifier string) Path {
	return &Identifier{
		identifier: identifier,
	}
}

func (this *Identifier) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIdentifier(this)
}

func (this *Identifier) Type() value.Type { return value.JSON }

func (this *Identifier) Evaluate(item value.Value, context Context) (value.Value, error) {
	rv, _ := item.Field(this.identifier)
	return rv, nil
}

func (this *Identifier) Alias() string {
	return this.identifier
}

func (this *Identifier) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *Identifier:
		return this.identifier == other.identifier
	default:
		return false
	}
}

func (this *Identifier) SubsetOf(other Expression) bool {
	return this.EquivalentTo(other)
}

func (this *Identifier) Children() Expressions {
	return nil
}

func (this *Identifier) MapChildren(mapper Mapper) error {
	return nil
}

func (this *Identifier) Set(item, val value.Value, context Context) bool {
	er := item.SetField(this.identifier, val)
	return er == nil
}

func (this *Identifier) Unset(item value.Value, context Context) bool {
	er := item.UnsetField(this.identifier)
	return er == nil
}

func (this *Identifier) Identifier() string {
	return this.identifier
}
