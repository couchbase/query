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

type Merge struct {
	keyspace  *KeyspaceRef           `json:"keyspace"`
	from      FromTerm               `json:"from"`
	query     *Select                `json:"select"`
	values    expression.Expressions `json:"values"`
	as        string                 `json:"as"`
	key       expression.Expression  `json:"key"`
	update    *MergeUpdate           `json:"update"`
	delete    *MergeDelete           `json:"delete"`
	insert    *MergeInsert           `json:"insert"`
	limit     expression.Expression  `json:"limit"`
	returning *Projection            `json:"returning"`
}

func NewMergeFrom(keyspace *KeyspaceRef, from FromTerm, as string, key expression.Expression,
	update *MergeUpdate, delete *MergeDelete, insert *MergeInsert,
	limit expression.Expression, returning *Projection) *Merge {
	return &Merge{
		keyspace:  keyspace,
		from:      from,
		query:     nil,
		values:    nil,
		as:        as,
		key:       key,
		update:    update,
		delete:    delete,
		insert:    insert,
		limit:     limit,
		returning: returning,
	}
}

func NewMergeSelect(keyspace *KeyspaceRef, query *Select, as string, key expression.Expression,
	update *MergeUpdate, delete *MergeDelete, insert *MergeInsert,
	limit expression.Expression, returning *Projection) *Merge {
	return &Merge{
		keyspace:  keyspace,
		from:      nil,
		query:     query,
		values:    nil,
		as:        as,
		key:       key,
		update:    update,
		delete:    delete,
		insert:    insert,
		limit:     limit,
		returning: returning,
	}
}

func NewMergeValues(keyspace *KeyspaceRef, values expression.Expressions, as string,
	key expression.Expression, update *MergeUpdate, delete *MergeDelete,
	insert *MergeInsert, limit expression.Expression, returning *Projection) *Merge {
	return &Merge{
		keyspace:  keyspace,
		from:      nil,
		query:     nil,
		values:    values,
		as:        as,
		key:       key,
		update:    update,
		delete:    delete,
		insert:    insert,
		limit:     limit,
		returning: returning,
	}
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

func (this *Merge) VisitExpressions(visitor expression.Visitor) (err error) {
	if this.from != nil {
		err = this.from.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.query != nil {
		err = this.query.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.values != nil {
		err = this.values.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.key != nil {
		expr, err := visitor.Visit(this.key)
		if err != nil {
			return err
		}

		this.key = expr.(expression.Expression)
	}

	if this.update != nil {
		err = this.update.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.delete != nil {
		err = this.delete.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.insert != nil {
		err = this.insert.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.limit != nil {
		expr, err := visitor.Visit(this.limit)
		if err != nil {
			return err
		}

		this.limit = expr.(expression.Expression)
	}

	if this.returning != nil {
		err = this.returning.VisitExpressions(visitor)
	}

	return
}

func (this *Merge) Formalize() (err error) {
	return
}

func (this *Merge) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

func (this *Merge) From() FromTerm {
	return this.from
}

func (this *Merge) Select() *Select {
	return this.query
}

func (this *Merge) Values() expression.Expressions {
	return this.values
}

func (this *Merge) As() string {
	return this.as
}

func (this *Merge) Key() expression.Expression {
	return this.key
}

func (this *Merge) Update() *MergeUpdate {
	return this.update
}

func (this *Merge) Delete() *MergeDelete {
	return this.delete
}

func (this *Merge) Insert() *MergeInsert {
	return this.insert
}

func (this *Merge) Limit() expression.Expression {
	return this.limit
}

func (this *Merge) Returning() *Projection {
	return this.returning
}

type MergeActions struct {
	Update *MergeUpdate
	Delete *MergeDelete
	Insert *MergeInsert
}

func NewMergeActions(update *MergeUpdate, delete *MergeDelete, insert *MergeInsert) *MergeActions {
	return &MergeActions{
		Update: update,
		Delete: delete,
		Insert: insert,
	}
}

type MergeUpdate struct {
	set   *Set                  `json:"set"`
	unset *Unset                `json:"unset"`
	where expression.Expression `json:"where"`
}

func NewMergeUpdate(set *Set, unset *Unset, where expression.Expression) *MergeUpdate {
	return &MergeUpdate{set, unset, where}
}

func (this *MergeUpdate) VisitExpressions(visitor expression.Visitor) (err error) {
	if this.set != nil {
		err = this.set.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.unset != nil {
		err = this.unset.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		expr, err := visitor.Visit(this.where)
		if err != nil {
			return err
		}

		this.where = expr.(expression.Expression)
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

func (this *MergeDelete) VisitExpressions(visitor expression.Visitor) (err error) {
	if this.where != nil {
		expr, err := visitor.Visit(this.where)
		if err != nil {
			return err
		}

		this.where = expr.(expression.Expression)
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

func (this *MergeInsert) VisitExpressions(visitor expression.Visitor) (err error) {
	if this.value != nil {
		expr, err := visitor.Visit(this.value)
		if err != nil {
			return err
		}

		this.value = expr.(expression.Expression)
	}

	if this.where != nil {
		expr, err := visitor.Visit(this.where)
		if err != nil {
			return err
		}

		this.where = expr.(expression.Expression)
	}

	return
}

func (this *MergeInsert) Value() expression.Expression {
	return this.value
}

func (this *MergeInsert) Where() expression.Expression {
	return this.where
}
