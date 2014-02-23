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
	bucket    *BucketRef            `json:"bucket"`
	from      FromTerm              `json:"from"`
	query     *Select               `json:"query"`
	as        string                `json:"as"`
	update    *MergeUpdate          `json:"update"`
	delete    *MergeDelete          `json:"delete"`
	insert    *MergeInsert          `json:"insert"`
	limit     expression.Expression `json:"limit"`
	returning ResultTerms           `json:"returning"`
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

func NewMerge(bucket *BucketRef, from FromTerm, query *Select, as string,
	update *MergeUpdate, delete *MergeDelete, insert *MergeInsert,
	limit expression.Expression, returning ResultTerms) *Merge {
	return &Merge{bucket, from, query, as, update,
		delete, insert, limit, returning}
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
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
