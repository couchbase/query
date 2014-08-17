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

func (this *Select) VisitExpressions(visitor expression.Visitor) (err error) {
	err = this.subresult.VisitExpressions(visitor)
	if err != nil {
		return
	}

	if this.order != nil {
		err = this.order.VisitExpressions(visitor)
	}

	return
}

func (this *Select) Formalize() (err error) {
	formalizer, err := this.subresult.Formalize()
	if err != nil {
		return err
	}

	if this.order != nil {
		err = this.order.VisitExpressions(formalizer)
		if err != nil {
			return
		}
	}

	return
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

func (this SortTerms) VisitExpressions(visitor expression.Visitor) (err error) {
	for _, term := range this {
		expr, err := visitor.Visit(term.expr)
		if err != nil {
			return err
		}

		term.expr = expr.(expression.Expression)
	}

	return
}

type Subresult interface {
	Node
	IsCorrelated() bool
	VisitExpressions(visitor expression.Visitor) error
	Formalize() (formalizer *expression.Formalizer, err error)
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

func (this *Subselect) VisitExpressions(visitor expression.Visitor) (err error) {
	if this.from != nil {
		err = this.from.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.let != nil {
		err = this.let.VisitExpressions(visitor)
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

	if this.group != nil {
		err = this.group.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	return this.projection.VisitExpressions(visitor)
}

func (this *Subselect) Formalize() (f *expression.Formalizer, err error) {
	f = &expression.Formalizer{}

	if this.from == nil {
		f.Allowed = value.EMPTY_OBJECT_VALUE
		return f, nil
	}

	f.Allowed, f.Keyspace, err = this.from.Formalize()
	if err != nil {
		return nil, err
	}

	if this.let != nil {
		f.Allowed, err = this.let.Formalize(f.Allowed, f.Keyspace)
		if err != nil {
			return nil, err
		}
	}

	if this.where != nil {
		expr, err := f.Visit(this.where)
		if err != nil {
			return nil, err
		}

		this.where = expr.(expression.Expression)
	}

	if this.group != nil {
		f.Allowed, err = this.group.Formalize(f.Allowed, f.Keyspace)
		if err != nil {
			return nil, err
		}
	}

	err = this.projection.VisitExpressions(f)
	if err != nil {
		return nil, err
	}

	return f, nil
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

func (this *Group) VisitExpressions(visitor expression.Visitor) (err error) {
	if this.by != nil {
		err = this.by.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.letting != nil {
		err = this.letting.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.having != nil {
		expr, err := visitor.Visit(this.having)
		if err != nil {
			return err
		}

		this.having = expr.(expression.Expression)
	}

	return
}

func (this *Group) Formalize(allowed value.Value, keyspace string) (a value.Value, err error) {
	if this.by != nil {
		for i, b := range this.by {
			this.by[i], err = b.Formalize(allowed, keyspace)
			if err != nil {
				return nil, err
			}
		}
	}

	if this.letting != nil {
		allowed, err = this.letting.Formalize(allowed, keyspace)
		if err != nil {
			return nil, err
		}
	}

	if this.having != nil {
		expr, err := this.having.Formalize(allowed, keyspace)
		if err != nil {
			return nil, err
		}

		this.having = expr.(expression.Expression)
	}

	return allowed, nil
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

func (this *Union) VisitExpressions(visitor expression.Visitor) (err error) {
	err = this.first.VisitExpressions(visitor)
	if err != nil {
		return
	}

	return this.second.VisitExpressions(visitor)
}

func (this *Union) Formalize() (f *expression.Formalizer, err error) {
	var ff, sf *expression.Formalizer
	ff, err = this.first.Formalize()
	if err != nil {
		return nil, err
	}

	sf, err = this.second.Formalize()
	if err != nil {
		return nil, err
	}

	// Intersection
	fa := ff.Allowed.Fields()
	sa := sf.Allowed.Fields()
	for field, _ := range fa {
		_, ok := sa[field]
		if !ok {
			delete(fa, field)
		}
	}

	ff.Allowed = value.NewValue(fa)
	return ff, nil
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

func (this *UnionAll) VisitExpressions(visitor expression.Visitor) (err error) {
	err = this.first.VisitExpressions(visitor)
	if err != nil {
		return
	}

	return this.second.VisitExpressions(visitor)
}

func (this *UnionAll) Formalize() (f *expression.Formalizer, err error) {
	var ff, sf *expression.Formalizer
	ff, err = this.first.Formalize()
	if err != nil {
		return nil, err
	}

	sf, err = this.second.Formalize()
	if err != nil {
		return nil, err
	}

	// Intersection
	fa := ff.Allowed.Fields()
	sa := sf.Allowed.Fields()
	for field, _ := range fa {
		_, ok := sa[field]
		if !ok {
			delete(fa, field)
		}
	}

	ff.Allowed = value.NewValue(fa)
	return ff, nil
}
