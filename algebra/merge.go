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
	bucket    *BucketRef             `json:"bucket"`
	from      FromTerm               `json:"from"`
	query     *Select                `json:"query"`
	values    expression.Expressions `json:"values"`
	as        string                 `json:"as"`
	key       expression.Expression  `json:"key"`
	update    *MergeUpdate           `json:"update"`
	delete    *MergeDelete           `json:"delete"`
	insert    *MergeInsert           `json:"insert"`
	limit     expression.Expression  `json:"limit"`
	returning *Projection            `json:"returning"`
}

func NewMergeSelect(bucket *BucketRef, query *Select, as string, key expression.Expression,
	update *MergeUpdate, delete *MergeDelete, insert *MergeInsert,
	limit expression.Expression, returning *Projection) *Merge {
	return &Merge{
		bucket:    bucket,
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

func NewMergeValues(bucket *BucketRef, values expression.Expressions, as string,
	key expression.Expression, update *MergeUpdate, delete *MergeDelete,
	insert *MergeInsert, limit expression.Expression, returning *Projection) *Merge {
	return &Merge{
		bucket:    bucket,
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

type MergeDelete struct {
	where expression.Expression `json:"where"`
}

type MergeInsert struct {
	value expression.Expression `json:"value"`
	where expression.Expression `json:"where"`
}

func NewMergeFrom(bucket *BucketRef, from FromTerm, as string, key expression.Expression,
	update *MergeUpdate, delete *MergeDelete, insert *MergeInsert,
	limit expression.Expression, returning *Projection) *Merge {
	return &Merge{
		bucket:    bucket,
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

func NewMergeUpdate(set *Set, unset *Unset, where expression.Expression) *MergeUpdate {
	return &MergeUpdate{set, unset, where}
}

func NewMergeDelete(where expression.Expression) *MergeDelete {
	return &MergeDelete{where}
}

func NewMergeInsert(value, where expression.Expression) *MergeInsert {
	return &MergeInsert{value, where}
}
