//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
SELECT statements can begin with either SELECT or FROM. The behavior
is the same in either case. The Subselect struct contains fields
mapping to each clause in the subselect statement. from, let, where,
group and projection, map to the FromTerm, let clause, group by
and select clause respectively.
*/
type Subselect struct {
	from        FromTerm              `json:"from"`
	let         expression.Bindings   `json:"let"`
	where       expression.Expression `json:"where"`
	group       *Group                `json:"group"`
	projection  *Projection           `json:"projection"`
	window      WindowTerms           `json:"window"`
	optimHints  *OptimHints           `json:"optimizer_hints"`
	correlated  bool                  `json:"correlated"`
	correlation map[string]bool       `json:"correlated_variables"`
}

/*
Constructor.
*/
func NewSubselect(from FromTerm, let expression.Bindings,
	where expression.Expression, group *Group, window WindowTerms,
	projection *Projection, optimHints *OptimHints) *Subselect {

	return &Subselect{
		from:       from,
		let:        let,
		where:      where,
		group:      group,
		projection: projection,
		window:     window,
		optimHints: optimHints,
	}
}

/*
Visitor pattern.
*/
func (this *Subselect) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitSubselect(this)
}

/*
This method returns the shape of the select clause. It returns a value
that represents the signature of the projection.
*/
func (this *Subselect) Signature() value.Value {
	return this.projection.Signature()
}

/*
This method qualifies identifiers for all the contituent
clauses namely the with, from, let, where, group and projection
in a subselect statement. It calls Formalize for the from,
group and projections, calls Map to map the where
expressions and calls PushBindings for the let clause.
*/
func (this *Subselect) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	if this.from != nil {
		f, err = this.from.Formalize(parent)
		if err != nil {
			return nil, err
		}
	} else {
		f = expression.NewFormalizer("", parent)
	}

	if this.let != nil {
		err = f.PushBindings(this.let, false)
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
		err = this.group.Formalize(f)
		if err != nil {
			return nil, err
		}
	}

	if this.window != nil {
		if err = this.window.Formalize(f); err != nil {
			return nil, err
		}

	}

	f, err = this.projection.Formalize(f)
	if err != nil {
		return nil, err
	}

	// Determine if this is a correlated subquery
	this.correlated = f.CheckCorrelated()
	if this.correlated {
		correlation := f.GetCorrelation()
		if this.correlation == nil {
			this.correlation = make(map[string]bool, len(correlation))
		}
		for k, v := range correlation {
			this.correlation[k] = v
		}
	}

	if this.from != nil && this.from.IsCorrelated() {
		correlation := this.from.GetCorrelation()
		if this.correlation == nil {
			this.correlation = make(map[string]bool, len(correlation))
		}
		for k, v := range correlation {
			if f.CheckCorrelation(k) {
				this.correlated = true
				this.correlation[k] = v
			}
		}
	}

	return f, nil
}

/*
This method maps all the constituent clauses, namely the from,
let, where, group by and projection(select) within a Subselect
statement.
*/
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

	if this.window != nil {
		if err = this.window.MapExpressions(mapper); err != nil {
			return
		}
	}

	return this.projection.MapExpressions(mapper)
}

/*
   Returns all contained Expressions.
*/
func (this *Subselect) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)

	if this.from != nil {
		exprs = append(exprs, this.from.Expressions()...)
	}

	if this.let != nil {
		exprs = append(exprs, this.let.Expressions()...)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	if this.group != nil {
		exprs = append(exprs, this.group.Expressions()...)
	}

	if this.window != nil {
		exprs = append(exprs, this.window.Expressions()...)
	}

	exprs = append(exprs, this.projection.Expressions()...)
	return exprs
}

/*
Result terms.
*/
func (this *Subselect) ResultTerms() ResultTerms {
	return this.projection.Terms()
}

/*
Raw projection.
*/
func (this *Subselect) Raw() bool {
	return this.projection.Raw()
}

/*
Returns all required privileges.
*/
func (this *Subselect) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()

	if this.from != nil {
		fprivs, err := this.from.Privileges()
		if err != nil {
			return nil, err
		}

		privs.AddAll(fprivs)
	}

	exprs := make(expression.Expressions, 0, 16)

	if this.let != nil {
		exprs = append(exprs, this.let.Expressions()...)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	if this.group != nil {
		exprs = append(exprs, this.group.Expressions()...)
	}

	if this.window != nil {
		exprs = append(exprs, this.window.Expressions()...)
	}

	exprs = append(exprs, this.projection.Expressions()...)

	subprivs, err := subqueryPrivileges(exprs)
	if err != nil {
		return nil, err
	}
	privs.AddAll(subprivs)

	for _, expr := range exprs {
		privs.AddAll(expr.Privileges())
	}

	return privs, nil
}

/*
   Representation as a N1QL string.
*/
func (this *Subselect) String() string {
	s := "select " + this.projection.String()

	if this.from != nil {
		s += " from " + this.from.String()
	}

	if this.let != nil {
		s += " let " + stringBindings(this.let)
	}

	if this.where != nil {
		s += " where " + this.where.String()
	}

	if this.group != nil {
		s += " " + this.group.String()
	}
	if this.window != nil {
		s += " " + this.window.String()
	}

	return s
}

func (this *Subselect) IsCorrelated() bool {
	return this.correlated
}

func (this *Subselect) SetCorrelated() {
	this.correlated = true
}

func (this *Subselect) GetCorrelation() map[string]bool {
	return this.correlation
}

/*
Returns a FromTerm that represents the From clause
in the subselect statement.
*/
func (this *Subselect) From() FromTerm {
	return this.from
}

/*
Returns the let field that represents the Let
clause in the subselect statement.
*/
func (this *Subselect) Let() expression.Bindings {
	return this.let
}

/*
Returns the where expression that represents the where
clause in the subselect statement.
*/
func (this *Subselect) Where() expression.Expression {
	return this.where
}

/*
Returns the group field that represents the group by
clause in the subselect statement.
*/
func (this *Subselect) Group() *Group {
	return this.group
}

/*
Returns the projection (select clause) in the subselect
statement.
*/
func (this *Subselect) Projection() *Projection {
	return this.projection
}

/*
Returns the Window in the subselect
statement.
*/
func (this *Subselect) Window() WindowTerms {
	return this.window
}

func (this *Subselect) ResetWindow() {
	this.window = nil
}

func (this *Subselect) OptimHints() *OptimHints {
	return this.optimHints
}

func (this *Subselect) SetOptimHints(optimHints *OptimHints) {
	this.optimHints = optimHints
}

func (this *Subselect) AddSubqueryTermHints(subqTermHints []*SubqOptimHints) {
	if len(subqTermHints) > 0 {
		if this.optimHints == nil {
			this.optimHints = NewOptimHints(nil, false)
		}
		this.optimHints.AddSubqTermHints(subqTermHints)
	}
}

/*
   Representation as a N1QL string.
*/
func stringBindings(bindings expression.Bindings) string {
	s := ""

	for i, b := range bindings {
		if i > 0 {
			s += ", "
		}

		s += "`"
		s += b.Variable()
		s += "` = "
		s += b.Expression().String()
	}

	return s
}

/*
   Representation as a N1QL WITH clause string.
*/

func withBindings(bindings expression.Bindings) string {
	s := " WITH "

	for i, b := range bindings {
		if i > 0 {
			s += ", "
		}

		s += "`" + b.Variable() + "` AS ( "
		s += b.Expression().String()
		s += " ) "
	}

	return s
}
