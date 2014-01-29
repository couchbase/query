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
	from       FromTerm             `json:"from"`
	where      Expression           `json:"where"`
	groupBy    ExpressionList       `json:"group_by"`
	having     Expression           `json:"having"`
	projection ResultExpressionList `json:"select"`
	distinct   bool                 `json:"distinct"`
	orderBy    SortExpressionList   `json:"orderby"`
	limit      Expression           `json:"limit"`
	offset     Expression           `json:"offset"`
}

type FromTerm interface {
	PrimaryTerm() *BucketTerm
}

type BucketTerm struct {
	pool       string
	bucket     string
	projection Path
	as         string
	keys       Expression
}

func NewBucketTerm(pool, bucket string, projection Path, as string, keys Expression) *BucketTerm {
	return &BucketTerm{pool, bucket, projection, as, keys}
}

func (this *BucketTerm) PrimaryTerm() *BucketTerm {
	return this
}

type Joiner int

const (
	JOIN Joiner = 1
	NEST        = 2
)

// For JOINs and NESTs
type Join struct {
	left   FromTerm
	outer  bool
	joiner Joiner
	right  *BucketTerm
}

func NewJoin(left FromTerm, outer bool, joiner Joiner, right *BucketTerm) *Join {
	return &Join{left, outer, joiner, right}
}

func (this *Join) PrimaryTerm() *BucketTerm {
	return this.left.PrimaryTerm()
}

type Unnest struct {
	left       FromTerm
	outer      bool
	projection Path
	as         string
}

func NewUnnest(left FromTerm, outer bool, projection Path, as string) *Unnest {
	return &Unnest{left, outer, projection, as}
}

func (this *Unnest) PrimaryTerm() *BucketTerm {
	return this.left.PrimaryTerm()
}

type ResultExpression struct {
	star bool       `json:"star"`
	expr Expression `json:"expr"`
	as   string     `json:"as"`
}

type ResultExpressionList []*ResultExpression

type SortExpression struct {
	expr      Expression `json:"expr"`
	ascending bool       `json:"asc"`
}

type SortExpressionList []*SortExpression

func NewSelect(from FromTerm, where Expression, groupBy ExpressionList,
	having Expression, projection ResultExpressionList, distinct bool,
	orderBy SortExpressionList, limit, offset Expression) *Select {
	return &Select{from, where, groupBy, having,
		projection, distinct, orderBy, limit, offset}
}

func (this *Select) VisitNode(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelect(this)
}
