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
	with       expression.Bindings   `json:"with"`
	from       FromTerm              `json:"from"`
	let        expression.Bindings   `json:"let"`
	where      expression.Expression `json:"where"`
	group      *Group                `json:"group"`
	projection *Projection           `json:"projection"`
	window     WindowTerms           `json:"window"`
	correlated bool                  `json:"correlated"`
}

/*
Constructor.
*/
func NewSubselect(with expression.Bindings, from FromTerm, let expression.Bindings,
	where expression.Expression, group *Group, window WindowTerms,
	projection *Projection) *Subselect {

	return &Subselect{
		with:       with,
		from:       from,
		let:        let,
		where:      where,
		group:      group,
		projection: projection,
		window:     window,
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
	if this.with != nil {
		f = expression.NewFormalizer("", parent)
		err = f.PushBindings(this.with, false)
		if err != nil {
			return nil, err
		}
		f.SetWiths(this.with)
	}

	if this.from != nil {
		if f == nil {
			f = parent
		}
		f, err = this.from.Formalize(f)
		if err != nil {
			return nil, err
		}
	} else if f == nil {
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
	this.correlated = false
	immediate := f.Allowed().GetValue().Fields()

	for ident, _ := range f.Identifiers().Fields() {
		if _, ok := immediate[ident]; !ok {
			if f.WithAlias(ident) {
				continue
			}
			this.correlated = true
			break
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
	var s string

	if len(this.with) > 0 {
		s += withBindings(this.with)
	}

	s += "select " + this.projection.String()

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

/*
Returns the let field that represents the With
clause in the subselect statement.
*/
func (this *Subselect) With() expression.Bindings {
	return this.with
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
