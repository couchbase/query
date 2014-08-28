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

func (this *Select) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.subresult.MapExpressions(mapper)
	if err != nil {
		return
	}

	if this.order != nil {
		err = this.order.MapExpressions(mapper)
	}

	return
}

func (this *Select) Formalize() (err error) {
	formalizer, err := this.subresult.Formalize()
	if err != nil {
		return err
	}

	if this.order != nil {
		err = this.order.MapExpressions(formalizer)
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

func (this SortTerms) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this {
		term.expr, err = mapper.Map(term.expr)
		if err != nil {
			return
		}
	}

	return
}

type Subresult interface {
	Node
	IsCorrelated() bool
	MapExpressions(mapper expression.Mapper) error
	Formalize() (formalizer *Formalizer, err error)
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

func (this *Subselect) MapExpressions(mapper expression.Mapper) (err error) {
	if this.from != nil {
		err = this.from.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.let != nil {
		err = this.let.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = mapper.Map(this.where)
		if err != nil {
			return
		}
	}

	if this.group != nil {
		err = this.group.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return this.projection.MapExpressions(mapper)
}

func (this *Subselect) Formalize() (f *Formalizer, err error) {
	if this.from == nil {
		f = NewFormalizer()
		return
	}

	f, err = this.from.Formalize()
	if err != nil {
		return
	}

	if this.let != nil {
		err = f.PushBindings(this.let)
		if err != nil {
			return nil, err
		}
	}

	if this.where != nil {
		expr, err := f.Map(this.where)
		if err != nil {
			return nil, err
		}

		this.where = expr
	}

	if this.group != nil {
		f, err = this.group.Formalize(f)
		if err != nil {
			return nil, err
		}
	}

	err = this.projection.MapExpressions(f)
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

func (this *Group) MapExpressions(mapper expression.Mapper) (err error) {
	if this.by != nil {
		err = this.by.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.letting != nil {
		err = this.letting.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.having != nil {
		this.having, err = mapper.Map(this.having)
	}

	return
}

func (this *Group) Formalize(f *Formalizer) (*Formalizer, error) {
	var err error

	if this.by != nil {
		for i, b := range this.by {
			this.by[i], err = f.Map(b)
			if err != nil {
				return nil, err
			}
		}
	}

	if this.letting != nil {
		err = f.PushBindings(this.letting)
		if err != nil {
			return nil, err
		}
	}

	if this.having != nil {
		this.having, err = f.Map(this.having)
		if err != nil {
			return nil, err
		}
	}

	return f, nil
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

func (this *binarySubresult) Formalize() (f *Formalizer, err error) {
	var ff, sf *Formalizer
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
	if ff.Keyspace != sf.Keyspace {
		ff.Keyspace = ""
	}

	return ff, nil
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

func (this *Union) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.first.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.second.MapExpressions(mapper)
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

func (this *UnionAll) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.first.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.second.MapExpressions(mapper)
}

type Intersect struct {
	binarySubresult
}

func NewIntersect(first, second Subresult) Subresult {
	return &Intersect{
		binarySubresult{
			first:  first,
			second: second,
		},
	}
}

func (this *Intersect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersect(this)
}

func (this *Intersect) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.first.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.second.MapExpressions(mapper)
}

type IntersectAll struct {
	binarySubresult
}

func NewIntersectAll(first, second Subresult) Subresult {
	return &IntersectAll{
		binarySubresult{
			first:  first,
			second: second,
		},
	}
}

func (this *IntersectAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntersectAll(this)
}

func (this *IntersectAll) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.first.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.second.MapExpressions(mapper)
}

type Except struct {
	binarySubresult
}

func NewExcept(first, second Subresult) Subresult {
	return &Except{
		binarySubresult{
			first:  first,
			second: second,
		},
	}
}

func (this *Except) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExcept(this)
}

func (this *Except) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.first.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.second.MapExpressions(mapper)
}

type ExceptAll struct {
	binarySubresult
}

func NewExceptAll(first, second Subresult) Subresult {
	return &ExceptAll{
		binarySubresult{
			first:  first,
			second: second,
		},
	}
}

func (this *ExceptAll) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExceptAll(this)
}

func (this *ExceptAll) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.first.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.second.MapExpressions(mapper)
}
