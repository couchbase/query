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

type Select struct {
	from     FromTerm    `json:"from"`
	where    Expression  `json:"where"`
	group    Expressions `json:"group"`
	having   Expression  `json:"having"`
	project  ResultTerms `json:"project"`
	distinct bool        `json:"distinct"`
	order    SortTerms   `json:"order"`
	offset   Expression  `json:"offset"`
	limit    Expression  `json:"limit"`
}

func NewSelect(from FromTerm, where Expression, group Expressions,
	having Expression, project ResultTerms, distinct bool,
	order SortTerms, offset Expression, limit Expression,
) *Select {
	return &Select{from, where, group, having,
		project, distinct, order, offset, limit}
}

func (this *Select) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelect(this)
}

func (this *Select) From() FromTerm {
	return this.from
}

func (this *Select) Where() Expression {
	return this.where
}

func (this *Select) Group() Expressions {
	return this.group
}

func (this *Select) Having() Expression {
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

func (this *Select) Offset() Expression {
	return this.offset
}

func (this *Select) Limit() Expression {
	return this.limit
}

func (this *Select) IsCorrelated() bool {
	return true // FIXME
}

type SortTerms []*SortTerm

type SortTerm struct {
	expr       Expression `json:"expr"`
	descending bool       `json:"asc"`
}

func (this *SortTerm) Expression() Expression {
	return this.expr
}

func (this *SortTerm) Descending() bool {
	return this.descending
}
