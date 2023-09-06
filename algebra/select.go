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

	subresult    Subresult             `json:"subresult"`
	with         expression.Bindings   `json:"with"`
	order        *Order                `json:"order"`
	offset       expression.Expression `json:"offset"`
	limit        expression.Expression `json:"limit"`
	correlated   bool                  `json:"correlated"`
	hasVariables bool                  `json:"hasVariables"` // not actually propagated
	inlineFunc   bool                  `json:"inlineFunction"`
	setop        bool                  `json:"setop"`
	correlation  map[string]uint32     `json:"correlation"`
}

/*
The function NewSelect returns a pointer to the Select struct
by assigning the input attributes to the fields of the struct.
*/
func NewSelect(subresult Subresult, with expression.Bindings, order *Order, offset, limit expression.Expression) *Select {
	rv := &Select{
		subresult: subresult,
		with:      with,
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

	if this.with != nil {
		err = this.with.MapExpressions(mapper)
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

	if this.with != nil {
		exprs = append(exprs, this.with.Expressions()...)
	}

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

	if this.with != nil {
		exprs = append(exprs, this.with.Expressions()...)
	}

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
	var s string

	if len(this.with) > 0 {
		s += withBindings(this.with)
	}

	s += this.subresult.String()

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
	if parent != nil && parent.InFunction() {
		this.inlineFunc = true
		if parent.HasVariables() {
			this.hasVariables = true
		}
	}
	if parent != nil && !this.setop {
		withs := parent.SaveWiths()
		defer parent.RestoreWiths(withs)
	}

	var f *expression.Formalizer
	if this.with != nil {
		f = expression.NewFormalizer("", parent)
		err = f.PushBindings(this.with, false)
		if err != nil {
			return err
		}
		f.SetWiths(this.with)
	} else {
		f = parent
	}

	f, err = this.subresult.Formalize(f)
	if err != nil {
		return err
	}

	this.correlated = this.subresult.IsCorrelated()
	if this.correlated {
		this.correlation = this.subresult.GetCorrelation()
	}

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

		correlated := f.CheckCorrelated()
		if correlated {
			correlation := f.GetCorrelation()
			this.correlated = correlated
			if this.correlation == nil {
				this.correlation = make(map[string]uint32, len(correlation))
			}
			for k, v := range correlation {
				this.correlation[k] |= v
			}
		}
	}

	if this.limit == nil && this.offset == nil {
		return err
	}

	prevIdentifiers := parent.Identifiers()
	defer parent.SetIdentifiers(prevIdentifiers)
	parent.SetIdentifiers(value.NewScopeValue(make(map[string]interface{}, 16), nil))

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

	fields := parent.Identifiers().Fields()
	if len(fields) > 0 {
		this.correlated = true
		if this.correlation == nil {
			this.correlation = make(map[string]uint32, len(fields))
		}
		for k, _ := range fields {
			this.correlation[k] |= expression.IDENT_IS_UNKNOWN
		}
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

func (this *Select) GetCorrelation() map[string]uint32 {
	return this.correlation
}

func (this *Select) HasVariables() bool {
	return this.hasVariables
}

func (this *Select) InInlineFunction() bool {
	return this.inlineFunc
}

/*
this.setop indicates whether the Select is under a set operation (UNION/INTERSECT/EXCEPT)
*/
func (this *Select) IsUnderSetOp() bool {
	return this.setop
}

func (this *Select) SetUnderSetOp() {
	this.setop = true
}

/*
Returns the With clause in the select statement.
*/
func (this *Select) With() expression.Bindings {
	return this.with
}

func (this *Select) OptimHints() *OptimHints {
	return this.subresult.OptimHints()
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
	   Get correlation
	*/
	GetCorrelation() map[string]uint32

	/*
	   Checks if projection is raw
	*/
	Raw() bool

	/*
	   Returns the optimizer hints
	*/
	OptimHints() *OptimHints
}
