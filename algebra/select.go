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

type Select struct {
	subresult Subresult             `json:"subresult"`
	order     SortTerms             `json:"order"`
	offset    expression.Expression `json:"offset"`
	limit     expression.Expression `json:"limit"`
}

func (this *Select) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelect(this)
}

func (this *Select) Subresult() Subresult {
	return this.subresult
}

func (this *Select) Order() SortTerms {
	return this.order
}

func (this *Select) Offset() expression.Expression {
	return this.offset
}

func (this *Select) Limit() expression.Expression {
	return this.limit
}

func (this *Select) SetLimit(limit expression.Expression) {
	this.limit = limit
}

type SortTerms []*SortTerm

type SortTerm struct {
	expr       expression.Expression `json:"expr"`
	descending bool                  `json:"desc"`
}

func NewSortTerm(expr expression.Expression, descending bool) *SortTerm {
	return &SortTerm{
		expr:       expr,
		descending: descending,
	}
}

func (this *SortTerm) Expression() expression.Expression {
	return this.expr
}

func (this *SortTerm) Descending() bool {
	return this.descending
}

type Subresult interface {
	Node
	IsCorrelated() bool
}

type Subselect struct {
	from     FromTerm               `json:"from"`
	let      expression.Bindings    `json:"let"`
	where    expression.Expression  `json:"where"`
	group    expression.Expressions `json:"group"`
	letting  expression.Bindings    `json:"letting"`
	having   expression.Expression  `json:"having"`
	project  ResultTerms            `json:"project"`
	distinct bool                   `json:"distinct"`
}

func NewSubselect(from FromTerm, let expression.Bindings, where expression.Expression,
	group expression.Expressions, letting expression.Bindings, having expression.Expression,
	project ResultTerms, distinct bool,
) *Subselect {
	return &Subselect{from, let, where, group, letting, having, project, distinct}
}

func (this *Subselect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSubselect(this)
}

func (this *Subselect) From() FromTerm {
	return this.from
}

func (this *Subselect) Where() expression.Expression {
	return this.where
}

func (this *Subselect) Group() expression.Expressions {
	return this.group
}

func (this *Subselect) Having() expression.Expression {
	return this.having
}

func (this *Subselect) Project() ResultTerms {
	return this.project
}

func (this *Subselect) Distinct() bool {
	return this.distinct
}

func (this *Subselect) IsCorrelated() bool {
	return true // FIXME
}

type binarySubresult struct {
	first  Subresult `json:"first"`
	second Subresult `json:"second"`
}

func (this *binarySubresult) IsCorrelated() bool {
	return this.first.IsCorrelated() || this.second.IsCorrelated()
}

func (this *binarySubresult) First() Subresult {
	return this.first
}

func (this *binarySubresult) Second() Subresult {
	return this.second
}

type Union struct {
	binarySubresult
}

func NewUnion(first, second Subresult) Subresult {
	return &Union{
		binarySubresult{
			first:  first,
			second: second,
		},
	}
}

func (this *Union) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnion(this)
}

type UnionAll struct {
	binarySubresult
}

func NewUnionAll(first, second Subresult) Subresult {
	return &UnionAll{
		binarySubresult{
			first:  first,
			second: second,
		},
	}
}

func (this *UnionAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnionAll(this)
}
