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

	subresult  Subresult             `json:"subresult"`
	order      *Order                `json:"order"`
	offset     expression.Expression `json:"offset"`
	limit      expression.Expression `json:"limit"`
	correlated bool                  `json:"correlated"`
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
Visitor pattern.
*/
func (this *Select) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelect(this)
}

/*
This method returns the shape of this statement.
*/
func (this *Select) Signature() value.Value {
	return this.subresult.Signature()
}

/*
It's a select
*/
func (this *Select) Type() string {
	return "SELECT"
}

/*
This method calls FormalizeSubquery to qualify all the children
of the query, and returns an error if any.
*/
func (this *Select) Formalize() (err error) {
	return this.FormalizeSubquery(expression.NewFormalizer("", nil))
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
func (this *Select) Privileges() (*auth.Privileges, errors.Error) {
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
	privs.AddAll(subprivs)

	for _, expr := range exprs {
		privs.AddAll(expr.Privileges())
	}

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
func (this *Select) FormalizeSubquery(parent *expression.Formalizer) error {
	if parent != nil {
		withs := parent.SaveWiths()
		defer parent.RestoreWiths(withs)
	}

	f, err := this.subresult.Formalize(parent)
	if err != nil {
		return err
	}

	this.correlated = this.subresult.IsCorrelated()

	if this.order != nil {
		err = this.order.MapExpressions(f)
		if err != nil {
			return err
		}

		// references to projection alias should be properly marked
		resultTerms := this.subresult.ResultTerms()
		aliases := make(map[string]bool, len(resultTerms))
		for _, pterm := range resultTerms {
			if !pterm.Star() && pterm.As() != "" {
				aliases[pterm.As()] = true
			}
		}
		if len(aliases) > 0 {
			for _, oterm := range this.order.Terms() {
				oterm.Expression().SetIdentFlags(aliases, expression.IDENT_IS_PROJ_ALIAS)
			}
		}

		if !this.correlated {
			// Determine if this is a correlated subquery
			immediate := f.Allowed().GetValue().Fields()
			for ident, _ := range f.Identifiers().Fields() {
				if _, ok := immediate[ident]; !ok {
					this.correlated = true
					break
				}
			}
		}
	}

	if this.limit == nil && this.offset == nil {
		return err
	}

	if !this.correlated {
		prevIdentifiers := parent.Identifiers()
		defer parent.SetIdentifiers(prevIdentifiers)
		parent.SetIdentifiers(value.NewScopeValue(make(map[string]interface{}, 16), nil))
	}

	if this.limit != nil {
		_, err = this.limit.Accept(parent)
		if err != nil {
			return err
		}
	}

	if this.offset != nil {
		_, err = this.offset.Accept(parent)
		if err != nil {
			return err
		}
	}

	if !this.correlated {
		this.correlated = len(parent.Identifiers().Fields()) > 0
	}

	return err
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
Sets the limit expression for the select statement.
*/
func (this *Select) SetLimit(limit expression.Expression) {
	this.limit = limit
}

func (this *Select) IsCorrelated() bool {
	return this.correlated
}

func (this *Select) SetCorrelated() {
	this.correlated = true
}

func (this *Select) EstResultSize() int64 {
	return this.subresult.EstResultSize()
}

/*
The Subresult interface represents the intermediate result of a
select statement. It inherits from Node.
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
	   Result terms.
	*/
	ResultTerms() ResultTerms

	/*
	   Returns all required privileges.
	*/
	Privileges() (*auth.Privileges, errors.Error)

	/*
	   Representation as a N1QL string.
	*/
	String() string

	/*
	   Checks if correlated subquery.
	*/
	IsCorrelated() bool

	/*
	   Checks if projection is raw
	*/
	Raw() bool

	/*
	   Estimated result size
	*/
	EstResultSize() int64
}
