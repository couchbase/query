//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbaselabs/query/expression"
)

type Update struct {
	keyspace  *KeyspaceRef          `json:"keyspace"`
	keys      expression.Expression `json:"keys"`
	set       *Set                  `json:"set"`
	unset     *Unset                `json:"unset"`
	where     expression.Expression `json:"where"`
	limit     expression.Expression `json:"limit"`
	returning *Projection           `json:"returning"`
}

func NewUpdate(keyspace *KeyspaceRef, keys expression.Expression, set *Set, unset *Unset,
	where, limit expression.Expression, returning *Projection) *Update {
	return &Update{keyspace, keys, set, unset, where, limit, returning}
}

func (this *Update) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdate(this)
}

func (this *Update) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

func (this *Update) Keys() expression.Expression {
	return this.keys
}

func (this *Update) Set() *Set {
	return this.set
}

func (this *Update) Unset() *Unset {
	return this.unset
}

func (this *Update) Where() expression.Expression {
	return this.where
}

func (this *Update) Limit() expression.Expression {
	return this.limit
}

func (this *Update) Returning() *Projection {
	return this.returning
}

type Set struct {
	terms SetTerms
}

func NewSet(terms SetTerms) *Set {
	return &Set{terms}
}

func (this *Set) Terms() SetTerms {
	return this.terms
}

type Unset struct {
	terms UnsetTerms
}

func NewUnset(terms UnsetTerms) *Unset {
	return &Unset{terms}
}

func (this *Unset) Terms() UnsetTerms {
	return this.terms
}

type SetTerms []*SetTerm

type SetTerm struct {
	path      expression.Path       `json:"path"`
	value     expression.Expression `json:"value"`
	updateFor *UpdateFor            `json:"path-for"`
}

func NewSetTerm(path expression.Path, value expression.Expression, updateFor *UpdateFor) *SetTerm {
	return &SetTerm{path, value, updateFor}
}

func (this *SetTerm) Path() expression.Path {
	return this.path
}

func (this *SetTerm) Value() expression.Expression {
	return this.value
}

func (this *SetTerm) UpdateFor() *UpdateFor {
	return this.updateFor
}

type UnsetTerms []*UnsetTerm

type UnsetTerm struct {
	path      expression.Path `json:"path"`
	updateFor *UpdateFor      `json:"path-for"`
}

func NewUnsetTerm(path expression.Path, updateFor *UpdateFor) *UnsetTerm {
	return &UnsetTerm{path, updateFor}
}

func (this *UnsetTerm) Path() expression.Path {
	return this.path
}

func (this *UnsetTerm) UpdateFor() *UpdateFor {
	return this.updateFor
}

type UpdateFor struct {
	bindings expression.Bindings
	when     expression.Expression
}

func NewUpdateFor(bindings expression.Bindings, when expression.Expression) *UpdateFor {
	return &UpdateFor{bindings, when}
}

func (this *UpdateFor) Bindings() expression.Bindings {
	return this.bindings
}

func (this *UpdateFor) When() expression.Expression {
	return this.when
}
