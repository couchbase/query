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
	"github.com/couchbaselabs/query/value"
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

func (this *Update) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

func (this *Update) MapExpressions(mapper expression.Mapper) (err error) {
	if this.keys != nil {
		this.keys, err = mapper.Map(this.keys)
		if err != nil {
			return
		}
	}

	if this.set != nil {
		err = this.set.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.unset != nil {
		err = this.unset.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = mapper.Map(this.where)
		if err != nil {
			return
		}
	}

	if this.limit != nil {
		this.limit, err = mapper.Map(this.limit)
		if err != nil {
			return
		}
	}

	if this.returning != nil {
		err = this.returning.MapExpressions(mapper)
	}

	return
}

func (this *Update) Formalize() (err error) {
	f, err := this.keyspace.Formalize()
	if err != nil {
		return err
	}

	if this.keys != nil {
		_, err = this.keys.Accept(expression.EMPTY_FORMALIZER)
		if err != nil {
			return
		}
	}

	if this.set != nil {
		err = this.set.Formalize(f)
		if err != nil {
			return
		}
	}

	if this.unset != nil {
		err = this.unset.Formalize(f)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = f.Map(this.where)
		if err != nil {
			return
		}
	}

	if this.limit != nil {
		_, err = this.limit.Accept(expression.EMPTY_FORMALIZER)
		if err != nil {
			return
		}
	}

	if this.returning != nil {
		err = this.returning.MapExpressions(f)
	}

	return
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

func (this *Set) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this.terms {
		err = term.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return
}

func (this *Set) Formalize(f *Formalizer) (err error) {
	for _, term := range this.terms {
		err = term.Formalize(f)
		if err != nil {
			return
		}
	}

	return
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

func (this *Unset) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this.terms {
		err = term.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return
}

func (this *Unset) Formalize(f *Formalizer) (err error) {
	for _, term := range this.terms {
		err = term.Formalize(f)
		if err != nil {
			return
		}
	}

	return
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

func (this *SetTerm) MapExpressions(mapper expression.Mapper) (err error) {
	path, err := mapper.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)

	this.value, err = mapper.Map(this.value)
	if err != nil {
		return
	}

	if this.updateFor != nil {
		err = this.updateFor.MapExpressions(mapper)
	}

	return
}

func (this *SetTerm) Formalize(f *Formalizer) (err error) {
	if this.updateFor != nil {
		defer f.PopBindings()
		err = f.PushBindings(this.updateFor.bindings)
		if err != nil {
			return
		}
	}

	path, err := f.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)
	this.value, err = f.Map(this.value)
	return
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

func (this *UnsetTerm) MapExpressions(mapper expression.Mapper) (err error) {
	path, err := mapper.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)

	if this.updateFor != nil {
		err = this.updateFor.MapExpressions(mapper)
	}

	return
}

func (this *UnsetTerm) Formalize(f *Formalizer) (err error) {
	if this.updateFor != nil {
		defer f.PopBindings()
		err = f.PushBindings(this.updateFor.bindings)
		if err != nil {
			return
		}
	}

	path, err := f.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)
	return
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

func (this *UpdateFor) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.bindings.MapExpressions(mapper)
	if err != nil {
		return
	}

	if this.when != nil {
		this.when, err = mapper.Map(this.when)
		if err != nil {
			return
		}
	}

	return
}

func (this *UpdateFor) Bindings() expression.Bindings {
	return this.bindings
}

func (this *UpdateFor) When() expression.Expression {
	return this.when
}
