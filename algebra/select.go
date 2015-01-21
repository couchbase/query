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
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

/*
This represents the select statement. Type Select is a
struct that contains fields mapping to each clause in
a select statement. The subresult field maps to the
intermediate result interface for the select clause.
The order field maps to the order by clause, the offset
is an expression that maps to the offset clause and
similarly limit is an expression that maps to the limit
clause.
*/
type Select struct {
	statementBase

	subresult Subresult             `json:"subresult"`
	order     *Order                `json:"order"`
	offset    expression.Expression `json:"offset"`
	limit     expression.Expression `json:"limit"`
}

/*
The function NewSelect returns a pointer to the Select struct
by assigning the input attributes to the fields of the struct.
*/
func NewSelect(subresult Subresult, order *Order, offset, limit expression.Expression) *Select {
	rv := &Select{
		subresult: subresult,
		order:     order,
		offset:    offset,
		limit:     limit,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitSelect method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Select) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelect(this)
}

/*
This method returns the shape of the subresult. It returns a value
that represents the signature of the subresult.
*/
func (this *Select) Signature() value.Value {
	return this.subresult.Signature()
}

/*
This method calls FormalizeSubquery to qualify all the children
of the query, and returns an error if any.
*/
func (this *Select) Formalize() (err error) {
	return this.FormalizeSubquery(expression.NewFormalizer())
}

/*
This method maps all the constituent clauses, namely the subresult,
order, limit and offset within a Select statement.
*/
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

/*
   Returns all contained Expressions.
*/
func (this *Select) Expressions() expression.Expressions {
	exprs := this.subresult.Expressions()

	if this.order != nil {
		exprs = append(exprs, this.order.Expressions()...)
	}

	if this.limit != nil {
		exprs = append(exprs, this.limit)
	}

	if this.offset != nil {
		exprs = append(exprs, this.offset)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *Select) Privileges() (datastore.Privileges, errors.Error) {
	privs, err := this.subresult.Privileges()
	if err != nil {
		return nil, err
	}

	exprs := make(expression.Expressions, 0, 16)

	if this.order != nil {
		exprs = append(exprs, this.order.Expressions()...)
	}

	if this.limit != nil {
		exprs = append(exprs, this.limit)
	}

	if this.offset != nil {
		exprs = append(exprs, this.offset)
	}

	subprivs, err := subqueryPrivileges(exprs)
	if err != nil {
		return nil, err
	}

	privs.Add(subprivs)
	return privs, nil
}

/*
   Representation as a N1QL string.
*/
func (this *Select) String() string {
	s := this.subresult.String()

	if this.order != nil {
		s += " " + this.order.String()
	}

	if this.limit != nil {
		s += " limit " + this.limit.String()
	}

	if this.offset != nil {
		s += " offset " + this.offset.String()
	}

	return s
}

/*
This method qualifies identifiers for all the constituent clauses,
namely the subresult, order, limit and offset within a subquery.
For the subresult of the subquery, call Formalize, for the order
by clause call MapExpressions, for limit and offset call Accept.
*/
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

/*
Return the subresult of the select statement.
*/
func (this *Select) Subresult() Subresult {
	return this.subresult
}

/*
Return the order by clause in the select statement.
*/
func (this *Select) Order() *Order {
	return this.order
}

/*
Returns the offset expression in the select clause.
*/
func (this *Select) Offset() expression.Expression {
	return this.offset
}

/*
Returns the limit expression in the select clause.
*/
func (this *Select) Limit() expression.Expression {
	return this.limit
}

/*
This method sets the limit expression for the select
statement.
*/
func (this *Select) SetLimit(limit expression.Expression) {
	this.limit = limit
}

/*
The Subresult interface represents the intermediate result of a
select statement. It inherits from Node and contains methods.
*/
type Subresult interface {
	/*
	   Inherts Node. The Node interface represents a node in
	   the algebra tree (AST).
	*/
	Node

	/*
	   The shape of this statement's return values.
	*/
	Signature() value.Value

	/*
	   Fully qualify all identifiers in this statement.
	*/
	Formalize(parent *expression.Formalizer) (formalizer *expression.Formalizer, err error)

	/*
	   Apply a Mapper to all the expressions in this statement
	*/
	MapExpressions(mapper expression.Mapper) error

	/*
	   Returns all contained Expressions.
	*/
	Expressions() expression.Expressions

	/*
	   Returns all required privileges.
	*/
	Privileges() (datastore.Privileges, errors.Error)

	/*
	   Representation as a N1QL string.
	*/
	String() string

	/*
	   Checks if correlated subquery.
	*/
	IsCorrelated() bool
}

/*
SELECT statements can begin with either SELECT or FROM. The behavior
is the same in either case. The Subselect struct contains fields
mapping to each clause in the subselect statement. from, let, where,
group and projection, map to the FromTerm, let clause, group by
and select clause respectively.
*/
type Subselect struct {
	from       FromTerm              `json:"from"`
	let        expression.Bindings   `json:"let"`
	where      expression.Expression `json:"where"`
	group      *Group                `json:"group"`
	projection *Projection           `json:"projection"`
}

/*
The function NewSubSelect returns a pointer to the Subselect struct
by assigning the input attributes to the fields of the struct.
*/
func NewSubselect(from FromTerm, let expression.Bindings, where expression.Expression,
	group *Group, projection *Projection) *Subselect {
	return &Subselect{from, let, where, group, projection}
}

/*
It calls the VisitSubselect method by passing in the receiver to
and returns the interface. It is a visitor pattern.
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
clauses namely the from, let, where, group and projection
in a subselect statement.It calls Formalize for the from,
group and projections, calls Map to map the where
expressions and calls PushBindings for the let clause.
*/
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
		_, err = f.PushBindings(this.let)
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

	exprs = append(exprs, this.projection.Expressions()...)
	return exprs
}

/*
Returns all required privileges.
*/
func (this *Subselect) Privileges() (datastore.Privileges, errors.Error) {
	privs := datastore.NewPrivileges()

	if this.from != nil {
		fprivs, err := this.from.Privileges()
		if err != nil {
			return nil, err
		}

		privs.Add(fprivs)
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

	exprs = append(exprs, this.projection.Expressions()...)

	subprivs, err := subqueryPrivileges(exprs)
	if err != nil {
		return nil, err
	}

	privs.Add(subprivs)
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

	return s
}

/*
Returns bool value that depicts if query is correlated
or not.
*/
func (this *Subselect) IsCorrelated() bool {
	return true // FIXME
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
