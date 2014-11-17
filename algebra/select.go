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
	order     *Order                `json:"order"`
	offset    expression.Expression `json:"offset"`
	limit     expression.Expression `json:"limit"`
}

func NewSelect(subresult Subresult, order *Order, offset, limit expression.Expression) *Select {
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

func (this *Select) Signature() value.Value {
	return this.subresult.Signature()
}

func (this *Select) Formalize() (err error) {
	return this.FormalizeSubquery(expression.NewFormalizer())
}

func (this *Select) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.subresult.MapExpressions(mapper)
	if err != nil {
		return
	}

	if this.order != nil {
		err = this.order.MapExpressions(mapper)
	}

	if this.limit != nil {
		this.limit, err = mapper.Map(this.limit)
		if err != nil {
			return
		}
	}

	if this.offset != nil {
		this.offset, err = mapper.Map(this.offset)
	}

	return
}

func (this *Select) FormalizeSubquery(parent *expression.Formalizer) (err error) {
	formalizer, err := this.subresult.Formalize(parent)
	if err != nil {
		return err
	}

	if this.order != nil && formalizer.Keyspace != "" {
		err = this.order.MapExpressions(formalizer)
		if err != nil {
			return
		}
	}

	if this.limit != nil {
		_, err = this.limit.Accept(parent)
		if err != nil {
			return
		}
	}

	if this.offset != nil {
		_, err = this.offset.Accept(parent)
		if err != nil {
			return
		}
	}

	return
}

func (this *Select) Subresult() Subresult {
	return this.subresult
}

func (this *Select) Order() *Order {
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

type Subresult interface {
	Node
	Signature() value.Value
	Formalize(parent *expression.Formalizer) (formalizer *expression.Formalizer, err error)
	MapExpressions(mapper expression.Mapper) error
	IsCorrelated() bool
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

func (this *Subselect) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitSubselect(this)
}

func (this *Subselect) Signature() value.Value {
	return this.projection.Signature()
}

func (this *Subselect) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	if this.from != nil {
		f, err = this.from.Formalize(parent)
		if err != nil {
			return
		}
	} else {
		f = parent
	}

	if this.let != nil {
		err = f.PushBindings(this.let)
		if err != nil {
			return nil, err
		}
	}

	if this.where != nil {
		this.where, err = f.Map(this.where)
		if err != nil {
			return nil, err
		}
	}

	if this.group != nil {
		f, err = this.group.Formalize(f)
		if err != nil {
			return nil, err
		}
	}

	f, err = this.projection.Formalize(f)
	if err != nil {
		return nil, err
	}

	return f, nil
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

func (this *Subselect) IsCorrelated() bool {
	return true // FIXME
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
