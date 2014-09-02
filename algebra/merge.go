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

type Merge struct {
	keyspace  *KeyspaceRef          `json:"keyspace"`
	source    *MergeSource          `json:"source"`
	key       expression.Expression `json:"key"`
	actions   *MergeActions         `json:"actions"`
	limit     expression.Expression `json:"limit"`
	returning *Projection           `json:"returning"`
}

func NewMerge(keyspace *KeyspaceRef, source *MergeSource, key expression.Expression,
	actions *MergeActions, limit expression.Expression, returning *Projection) *Merge {
	return &Merge{
		keyspace:  keyspace,
		source:    source,
		key:       key,
		actions:   actions,
		limit:     limit,
		returning: returning,
	}
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

func (this *Merge) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

func (this *Merge) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.source.MapExpressions(mapper)
	if err != nil {
		return
	}

	this.key, err = mapper.Map(this.key)
	if err != nil {
		return
	}

	err = this.actions.MapExpressions(mapper)
	if err != nil {
		return
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

func (this *Merge) Formalize() (err error) {
	kf, err := this.keyspace.Formalize()
	if err != nil {
		return err
	}

	sf, err := this.source.Formalize()
	if err != nil {
		return err
	}

	this.key, err = sf.Map(this.key)
	if err != nil {
		return err
	}

	if this.limit != nil {
		_, err = this.limit.Accept(expression.EMPTY_FORMALIZER)
		if err != nil {
			return
		}
	}

	if this.returning != nil {
		err = this.returning.MapExpressions(kf)
	}

	return
}

func (this *Merge) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

func (this *Merge) Source() *MergeSource {
	return this.source
}

func (this *Merge) Key() expression.Expression {
	return this.key
}

func (this *Merge) Actions() *MergeActions {
	return this.actions
}

func (this *Merge) Limit() expression.Expression {
	return this.limit
}

func (this *Merge) Returning() *Projection {
	return this.returning
}

type MergeSource struct {
	from   FromTerm               `json:"from"`
	query  *Select                `json:"select"`
	values expression.Expressions `json:"values"`
	as     string                 `json:"as"`
}

func NewMergeSourceFrom(from FromTerm, as string) *MergeSource {
	return &MergeSource{
		from: from,
		as:   as,
	}
}

func NewMergeSourceSelect(query *Select, as string) *MergeSource {
	return &MergeSource{
		query: query,
		as:    as,
	}
}

func NewMergeSourceValues(values expression.Expressions, as string) *MergeSource {
	return &MergeSource{
		values: values,
		as:     as,
	}
}

func (this *MergeSource) MapExpressions(mapper expression.Mapper) (err error) {
	if this.query != nil {
		err = this.query.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.values != nil {
		err = this.values.MapExpressions(mapper)
	}

	return
}

func (this *MergeSource) Formalize() (f *Formalizer, err error) {
	return
}

func (this *MergeSource) From() FromTerm {
	return this.from
}

func (this *MergeSource) Select() *Select {
	return this.query
}

func (this *MergeSource) Values() expression.Expressions {
	return this.values
}

func (this *MergeSource) As() string {
	return this.as
}

type MergeActions struct {
	update *MergeUpdate `json:"update"`
	delete *MergeDelete `json:"delete"`
	insert *MergeInsert `json:"insert"`
}

func NewMergeActions(update *MergeUpdate, delete *MergeDelete, insert *MergeInsert) *MergeActions {
	return &MergeActions{
		update: update,
		delete: delete,
		insert: insert,
	}
}

func (this *MergeActions) MapExpressions(mapper expression.Mapper) (err error) {
	if this.update != nil {
		err = this.update.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.delete != nil {
		err = this.delete.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.insert != nil {
		err = this.insert.MapExpressions(mapper)
	}

	return
}

func (this *MergeActions) Update() *MergeUpdate {
	return this.update
}

func (this *MergeActions) Delete() *MergeDelete {
	return this.delete
}

func (this *MergeActions) Insert() *MergeInsert {
	return this.insert
}

type MergeUpdate struct {
	set   *Set                  `json:"set"`
	unset *Unset                `json:"unset"`
	where expression.Expression `json:"where"`
}

func NewMergeUpdate(set *Set, unset *Unset, where expression.Expression) *MergeUpdate {
	return &MergeUpdate{set, unset, where}
}

func (this *MergeUpdate) MapExpressions(mapper expression.Mapper) (err error) {
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
	}

	return
}

func (this *MergeUpdate) Set() *Set {
	return this.set
}

func (this *MergeUpdate) Unset() *Unset {
	return this.unset
}

func (this *MergeUpdate) Where() expression.Expression {
	return this.where
}

type MergeDelete struct {
	where expression.Expression `json:"where"`
}

func NewMergeDelete(where expression.Expression) *MergeDelete {
	return &MergeDelete{where}
}

func (this *MergeDelete) MapExpressions(mapper expression.Mapper) (err error) {
	if this.where != nil {
		this.where, err = mapper.Map(this.where)
	}

	return
}

func (this *MergeDelete) Where() expression.Expression {
	return this.where
}

type MergeInsert struct {
	value expression.Expression `json:"value"`
	where expression.Expression `json:"where"`
}

func NewMergeInsert(value, where expression.Expression) *MergeInsert {
	return &MergeInsert{value, where}
}

func (this *MergeInsert) MapExpressions(mapper expression.Mapper) (err error) {
	if this.value != nil {
		this.value, err = mapper.Map(this.value)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = mapper.Map(this.where)
	}

	return
}

func (this *MergeInsert) Value() expression.Expression {
	return this.value
}

func (this *MergeInsert) Where() expression.Expression {
	return this.where
}
