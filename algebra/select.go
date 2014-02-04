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
	_ "fmt"
	_ "github.com/couchbaselabs/query/value"
)

type Select struct {
	from     FromTerm       `json:"from"`
	where    Expression     `json:"where"`
	group    ExpressionList `json:"group"`
	having   Expression     `json:"having"`
	project  ResultTermList `json:"project"`
	distinct bool           `json:"distinct"`
	order    SortTermList   `json:"order"`
	offset   Expression     `json:"offset"`
	limit    Expression     `json:"limit"`
}

type FromTerm interface {
	Node
	PrimaryTerm() FromTerm
}

type BucketTerm struct {
	pool    string
	bucket  string
	project Path
	as      string
	keys    Expression
}

// For subqueries.
type ParentTerm struct {
	project Path
	as      string
}

type Join struct {
	left  FromTerm
	outer bool
	right *BucketTerm
}

type Nest struct {
	left  FromTerm
	outer bool
	right *BucketTerm
}

type Unnest struct {
	left       FromTerm
	outer      bool
	projection Path
	as         string
}

func NewSelect(from FromTerm, where Expression, group ExpressionList,
	having Expression, project ResultTermList, distinct bool,
	order SortTermList, offset Expression, limit Expression,
) *Select {
	return &Select{from, where, group, having,
		project, distinct, order, offset, limit}
}

func (this *Select) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelect(this)
}

func (this *Select) IsCorrelated() bool {
	return true // FIXME
}

func NewBucketTerm(pool, bucket string, project Path, as string, keys Expression) *BucketTerm {
	return &BucketTerm{pool, bucket, project, as, keys}
}

func (this *BucketTerm) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBucketTerm(this)
}

func (this *BucketTerm) PrimaryTerm() FromTerm {
	return this
}

func NewParentTerm(project Path, as string) *ParentTerm {
	return &ParentTerm{project, as}
}

func (this *ParentTerm) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParentTerm(this)
}

func (this *ParentTerm) PrimaryTerm() FromTerm {
	return this
}

func NewJoin(left FromTerm, outer bool, right *BucketTerm) *Join {
	return &Join{left, outer, right}
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

func NewNest(left FromTerm, outer bool, right *BucketTerm) *Nest {
	return &Nest{left, outer, right}
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

func NewUnnest(left FromTerm, outer bool, projection Path, as string) *Unnest {
	return &Unnest{left, outer, projection, as}
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

type SortTerm struct {
	expr      Expression `json:"expr"`
	ascending bool       `json:"asc"`
}

type SortTermList []*SortTerm
