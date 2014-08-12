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

type Select struct {
	subresult Subresult             `json:"subresult"`
	order     SortTerms             `json:"order"`
	offset    expression.Expression `json:"offset"`
	limit     expression.Expression `json:"limit"`
}

func NewSelect(subresult Subresult, order SortTerms, offset, limit expression.Expression) *Select {
	return &Select{
		subresult: subresult,
		order:     order,
		offset:    offset,
		limit:     limit,
	}
}

func (this *Select) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelect(this)
}

func (this *Select) Formalize() (*Select, error) {
	subresult, forbidden, allowed, keyspace, err := this.subresult.Formalize()
	if err != nil {
		return nil, err
	}

	order := this.order
	if order != nil {
		order, err = order.Formalize(forbidden, allowed, keyspace)
		if err != nil {
			return nil, err
		}
	}

	return NewSelect(subresult, order, this.offset, this.limit), nil
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

func (this SortTerms) Formalize(forbidden, allowed value.Value, keyspace string) (
	sortTerms SortTerms, err error) {
	sortTerms = make(SortTerms, len(this))
	for i, term := range this {
		sortTerms[i] = &SortTerm{
			descending: term.descending,
		}

		sortTerms[i].expr, err = term.expr.Formalize(forbidden, allowed, keyspace)
		if err != nil {
			return nil, err
		}
	}

	return
}

type Subresult interface {
	Node
	IsCorrelated() bool
	Formalize() (subresult Subresult, forbidden, allowed value.Value, keyspace string, err error)
}

type Subselect struct {
	from       FromTerm              `json:"from"`
	let        expression.Bindings   `json:"let"`
	where      expression.Expression `json:"where"`
	group      *Group                `json:"group"`
	projection *Projection           `json:"projection"`
}

func NewSubselect(from FromTerm, let expression.Bindings, where expression.Expression,
	group *Group, projection *Projection) *Subselect {
	return &Subselect{from, let, where, group, projection}
}

func (this *Subselect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSubselect(this)
}

func (this *Subselect) Formalize() (subresult Subresult, forbidden, allowed value.Value, keyspace string, err error) {
	if this.from == nil {
		forbidden = value.EMPTY_OBJECT_VALUE
		allowed = value.EMPTY_OBJECT_VALUE
		return this, forbidden, allowed, "", nil
	}

	forbidden, allowed, keyspace, err = this.from.Formalize()
	if err != nil {
		return nil, nil, nil, "", err
	}

	let := this.let
	if let != nil {
		let, forbidden, allowed, err = this.let.Formalize(forbidden, allowed, keyspace)
		if err != nil {
			return nil, nil, nil, "", err
		}
	}

	where := this.where
	if where != nil {
		where, err = this.where.Formalize(forbidden, allowed, keyspace)
		if err != nil {
			return nil, nil, nil, "", err
		}
	}

	group := this.group
	if group != nil {
		group, forbidden, allowed, err = this.group.Formalize(forbidden, allowed, keyspace)
		if err != nil {
			return nil, nil, nil, "", err
		}
	}

	projection, err := this.projection.Formalize(forbidden, allowed, keyspace)
	if err != nil {
		return nil, nil, nil, "", err
	}

	return NewSubselect(this.from, let, where, group, projection), forbidden, allowed, keyspace, nil
}

func (this *Subselect) From() FromTerm {
	return this.from
}

func (this *Subselect) Let() expression.Bindings {
	return this.let
}

func (this *Subselect) Where() expression.Expression {
	return this.where
}

func (this *Subselect) Group() *Group {
	return this.group
}

func (this *Subselect) Projection() *Projection {
	return this.projection
}

func (this *Subselect) IsCorrelated() bool {
	return true // FIXME
}

type Group struct {
	by      expression.Expressions `json:by`
	letting expression.Bindings    `json:"letting"`
	having  expression.Expression  `json:"having"`
}

func NewGroup(by expression.Expressions, letting expression.Bindings, having expression.Expression) *Group {
	return &Group{
		by:      by,
		letting: letting,
		having:  having,
	}
}

func (this *Group) Formalize(forbidden, allowed value.Value, keyspace string) (
	group *Group, f, a value.Value, err error) {
	by := this.by
	if by != nil {
		by = make(expression.Expressions, len(this.by))
		for i, b := range this.by {
			by[i], err = b.Formalize(forbidden, allowed, keyspace)
			if err != nil {
				return nil, nil, nil, err
			}
		}
	}

	letting := this.letting
	if letting != nil {
		letting, f, a, err = this.letting.Formalize(forbidden, allowed, keyspace)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		f = forbidden
		a = allowed
	}

	having := this.having
	if having != nil {
		having, err = this.having.Formalize(f, a, keyspace)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return NewGroup(by, letting, having), f, a, nil
}

func (this *Group) By() expression.Expressions {
	return this.by
}

func (this *Group) Letting() expression.Bindings {
	return this.letting
}

func (this *Group) Having() expression.Expression {
	return this.having
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

func (this *Union) Formalize() (subresult Subresult, forbidden, allowed value.Value, keyspace string, err error) {
	first, _, fa, _, err := this.first.Formalize()
	if err != nil {
		return nil, nil, nil, "", err
	}

	second, _, sa, _, err := this.second.Formalize()
	if err != nil {
		return nil, nil, nil, "", err
	}

	// Intersection
	ff := fa.Fields()
	sf := sa.Fields()
	for f, _ := range ff {
		_, ok := sf[f]
		if !ok {
			delete(ff, f)
		}
	}

	allowed = value.NewValue(ff)
	return NewUnion(first, second), nil, allowed, "", nil
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

func (this *UnionAll) Formalize() (subresult Subresult, forbidden, allowed value.Value, keyspace string, err error) {
	first, _, fa, _, err := this.first.Formalize()
	if err != nil {
		return nil, nil, nil, "", err
	}

	second, _, sa, _, err := this.second.Formalize()
	if err != nil {
		return nil, nil, nil, "", err
	}

	// Intersection
	ff := fa.Fields()
	sf := sa.Fields()
	for f, _ := range ff {
		_, ok := sf[f]
		if !ok {
			delete(ff, f)
		}
	}

	allowed = value.NewValue(ff)
	return NewUnionAll(first, second), nil, allowed, "", nil
}
