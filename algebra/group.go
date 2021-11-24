//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/expression"
)

/*
This represents the Group by clause. Type Group is a
struct that contains group by expression 'by', the
letting clause and the having clause represented by
expression bindings and expressions respectively.
Aliases in the LETTING clause create new names that
may be referred to in the HAVING, SELECT, and ORDER
BY clauses. Having specifies a condition.
*/
type Group struct {
	by      expression.Expressions `json:by`
	letting expression.Bindings    `json:"letting"`
	having  expression.Expression  `json:"having"`
}

/*
The function NewGroup returns a pointer to the Group
struct that has its field sort terms set to the input
argument expressions.
*/
func NewGroup(by GroupTerms, letting expression.Bindings, having expression.Expression) *Group {
	rv := &Group{
		by:     by.Expressions(),
		having: having,
	}

	var byAlias expression.Bindings
	for _, g := range by {
		if g.As() != "" {
			byAlias = append(byAlias, expression.NewSimpleBinding(g.As(), g.Expression()))
		}
	}

	rv.letting = append(byAlias, letting...)
	return rv
}

/*
This method qualifies identifiers for all the constituent clauses,
namely the by, letting and having expressions by mapping them.
*/
func (this *Group) Formalize(f *expression.Formalizer) error {
	var err error

	if this.by != nil {
		for i, b := range this.by {
			this.by[i], err = f.Map(b)
			if err != nil {
				return err
			}
		}
	}

	if this.letting != nil {
		err = f.PushBindings(this.letting, false)
		if err != nil {
			return err
		}
	}

	if this.having != nil {
		this.having, err = f.Map(this.having)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
This method maps all the constituent clauses, namely the
by, letting and having within a group by clause.
*/
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

/*
   Returns all contained Expressions.
*/
func (this *Group) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)

	if this.by != nil {
		exprs = append(exprs, this.by...)
	}

	if this.letting != nil {
		exprs = append(exprs, this.letting.Expressions()...)
	}

	if this.having != nil {
		exprs = append(exprs, this.having)
	}

	return exprs
}

/*
   Representation as a N1QL string.
*/
func (this *Group) String() string {
	s := ""

	if this.by != nil {
		s += " group by "

		for i, b := range this.by {
			if i > 0 {
				s += ", "
			}

			s += b.String()
		}
	}

	if this.letting != nil {
		s += " letting " + stringBindings(this.letting)
	}

	if this.having != nil {
		s += " having " + this.having.String()
	}

	return s
}

/*
Returns the Group by expression.
*/
func (this *Group) By() expression.Expressions {
	return this.by
}

/*
Returns the letting expression bindings.
*/
func (this *Group) Letting() expression.Bindings {
	return this.letting
}

/*
Returns the having condition expression.
*/
func (this *Group) Having() expression.Expression {
	return this.having
}

type GroupTerms []*GroupTerm

func (this GroupTerms) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, len(this))

	for i, b := range this {
		exprs[i] = b.expr
	}

	return exprs
}

type GroupTerm struct {
	expr expression.Expression `json:"expr"`
	as   string                `json:"as"`
}

func NewGroupTerm(expr expression.Expression, as string) *GroupTerm {
	return &GroupTerm{
		expr: expr,
		as:   as,
	}
}

func (this *GroupTerm) MapExpression(mapper expression.Mapper) (err error) {
	if this.expr != nil {
		this.expr, err = mapper.Map(this.expr)
	}

	return
}

func (this *GroupTerm) String() string {
	s := ""

	if this.expr != nil {
		s = this.expr.String()
	}

	if this.as != "" {
		s += " as `" + this.as + "`"
	}

	return s
}

func (this *GroupTerm) Expression() expression.Expression {
	return this.expr
}

func (this *GroupTerm) As() string {
	return this.as
}
