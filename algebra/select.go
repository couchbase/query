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
	from     FromTerm               `json:"from"`
	let      Bindings               `json:"let"`
	where    expression.Expression  `json:"where"`
	group    expression.Expressions `json:"group"`
	letting  Bindings               `json:"letting"`
	having   expression.Expression  `json:"having"`
	project  ResultTerms            `json:"project"`
	distinct bool                   `json:"distinct"`
	order    SortTerms              `json:"order"`
	offset   expression.Expression  `json:"offset"`
	limit    expression.Expression  `json:"limit"`
}

func NewSelect(from FromTerm, let Bindings, where expression.Expression, group expression.Expressions,
	letting Bindings, having expression.Expression, project ResultTerms, distinct bool,
	order SortTerms, offset expression.Expression, limit expression.Expression,
) *Select {
	return &Select{from, let, where, group, letting, having,
		project, distinct, order, offset, limit}
}

func (this *Select) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelect(this)
}

func (this *Select) From() FromTerm {
	return this.from
}

func (this *Select) Where() expression.Expression {
	return this.where
}

func (this *Select) Group() expression.Expressions {
	return this.group
}

func (this *Select) Having() expression.Expression {
	return this.having
}

func (this *Select) Project() ResultTerms {
	return this.project
}

func (this *Select) Distinct() bool {
	return this.distinct
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

func (this *Select) IsCorrelated() bool {
	return true // FIXME
}

type SortTerms []*SortTerm

type SortTerm struct {
	expr       expression.Expression `json:"expr"`
	descending bool                  `json:"asc"`
}

func (this *SortTerm) Expression() expression.Expression {
	return this.expr
}

func (this *SortTerm) Descending() bool {
	return this.descending
}
