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
)

type FromTerm interface {
	Node
	PrimaryTerm() FromTerm
	Alias() string
}

type BucketTerm struct {
	pool    string
	bucket  string
	project Path
	as      string
	keys    Expression
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

func (this *BucketTerm) Alias() string {
	if this.as != "" {
		return this.as
	} else if this.project != nil {
		return this.project.Alias()
	} else {
		return this.bucket
	}
}

func (this *BucketTerm) Pool() string {
	return this.pool
}

func (this *BucketTerm) Bucket() string {
	return this.bucket
}

func (this *BucketTerm) Project() Path {
	return this.project
}

func (this *BucketTerm) As() string {
	return this.as
}

func (this *BucketTerm) Keys() Expression {
	return this.keys
}

// For subqueries.
type ParentTerm struct {
	project Path
	as      string
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

func (this *ParentTerm) Alias() string {
	return this.as
}

func (this *ParentTerm) Project() Path {
	return this.project
}

func (this *ParentTerm) As() string {
	return this.as
}

type Join struct {
	left  FromTerm
	right *BucketTerm
	outer bool
}

func NewJoin(left FromTerm, outer bool, right *BucketTerm) *Join {
	return &Join{left, right, outer}
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

func (this *Join) Alias() string {
	return this.right.Alias()
}

func (this *Join) Left() FromTerm {
	return this.left
}

func (this *Join) Right() *BucketTerm {
	return this.right
}

func (this *Join) Outer() bool {
	return this.outer
}

type Nest struct {
	left  FromTerm
	right *BucketTerm
	outer bool
}

func NewNest(left FromTerm, outer bool, right *BucketTerm) *Nest {
	return &Nest{left, right, outer}
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

func (this *Nest) Alias() string {
	return this.right.Alias()
}

func (this *Nest) Left() FromTerm {
	return this.left
}

func (this *Nest) Right() *BucketTerm {
	return this.right
}

func (this *Nest) Outer() bool {
	return this.outer
}

type Unnest struct {
	left    FromTerm
	outer   bool
	project Path
	as      string
}

func NewUnnest(left FromTerm, outer bool, project Path, as string) *Unnest {
	return &Unnest{left, outer, project, as}
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

func (this *Unnest) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.project.Alias()
	}
}

func (this *Unnest) Left() FromTerm {
	return this.left
}

func (this *Unnest) Outer() bool {
	return this.outer
}

func (this *Unnest) Project() Path {
	return this.project
}

func (this *Unnest) As() string {
	return this.as
}
