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
)

type SubqueryTerm struct {
	subquery     *Select
	as           string
	joinHint     JoinHint
	property     uint32
	correlation  map[string]uint32
	errorContext expression.ErrorContext
}

/*
Constructor.
*/
func NewSubqueryTerm(subquery *Select, as string, joinHint JoinHint) *SubqueryTerm {
	return &SubqueryTerm{
		subquery: subquery,
		as:       as,
		joinHint: joinHint,
	}
}

/*
Visitor pattern.
*/
func (this *SubqueryTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitSubqueryTerm(this)
}

/*
Apply mapping to all contained Expressions.
*/
func (this *SubqueryTerm) MapExpressions(mapper expression.Mapper) (err error) {
	return this.subquery.MapExpressions(mapper)
}

/*
Returns all contained Expressions.
*/
func (this *SubqueryTerm) Expressions() expression.Expressions {
	return this.subquery.Expressions()
}

/*
Returns all required privileges.
*/
func (this *SubqueryTerm) Privileges() (*auth.Privileges, errors.Error) {
	return this.subquery.Privileges()
}

/*
Representation as a N1QL string.
*/
func (this *SubqueryTerm) String() string {
	var s string

	if this.subquery.IsCorrelated() || this.subquery.subresult.IsCorrelated() {
		s += "correlated "
	}

	s += "(" + this.subquery.String() + ") as " + this.as
	if js := this.joinHint.String(); len(js) > 0 {
		s += " use" + js + " "
	}
	return s
}

/*
Qualify all identifiers for the parent expression. Checks for
duplicate aliases.
*/
func (this *SubqueryTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	alias := this.Alias()
	if alias == "" {
		err = errors.NewNoTermNameError("FROM Subquery", this.errorContext.String(), "semantics.subquery.requires_name_or_alias")
		return
	}

	if ok := parent.AllowedAlias(alias, true, false); ok {
		err = errors.NewDuplicateAliasError("subquery", alias, this.errorContext.String(), "semantics.subquery.duplicate_alias")
		return nil, err
	}

	f1 := expression.NewFormalizer("", parent)
	err = this.subquery.FormalizeSubquery(f1, true)
	if err != nil {
		return
	}

	if this.subquery.IsCorrelated() {
		this.correlation = addSimpleTermCorrelation(this.correlation,
			this.subquery.GetCorrelation(), this.IsAnsiJoinOp(), parent)
		checkLateralCorrelation(this)
	}

	// for checking subquery we need a new formalizer, however, if this SubqueryTerm
	// is under an ANSI join/nest operation we need to use the parent's formalizer
	if this.IsAnsiJoinOp() {
		f = parent
		f.SetKeyspace("")
	} else {
		f = f1
		f.SetExprSubqKeyspace(alias)
	}
	f.SetAllowedSubqTermAlias(alias)
	f.SetAlias(this.as)
	return
}

/*
Return the primary term in the from clause.
*/
func (this *SubqueryTerm) PrimaryTerm() SimpleFromTerm {
	return this
}

/*
Returns the Alias string.
*/
func (this *SubqueryTerm) Alias() string {
	return this.as
}

/*
Returns the inner subquery.
*/
func (this *SubqueryTerm) Subquery() *Select {
	return this.subquery
}

/*
Returns the join hint
*/
func (this *SubqueryTerm) JoinHint() JoinHint {
	return this.joinHint
}

/*
Join hint prefers hash join
*/
func (this *SubqueryTerm) PreferHash() bool {
	return this.joinHint == USE_HASH_BUILD || this.joinHint == USE_HASH_PROBE || this.joinHint == USE_HASH_EITHER
}

/*
Join hint prefers nested loop join
*/
func (this *SubqueryTerm) PreferNL() bool {
	return this.joinHint == USE_NL
}

/*
Returns the property.
*/
func (this *SubqueryTerm) Property() uint32 {
	return this.property
}

/*
Returns whether this subquery term is for an ANSI JOIN
*/
func (this *SubqueryTerm) IsAnsiJoin() bool {
	return (this.property & TERM_ANSI_JOIN) != 0
}

/*
Returns whether this subquery term is for an ANSI NEST
*/
func (this *SubqueryTerm) IsAnsiNest() bool {
	return (this.property & TERM_ANSI_NEST) != 0
}

/*
Returns whether this subquery term is for an ANSI JOIN or ANSI NEST
*/
func (this *SubqueryTerm) IsAnsiJoinOp() bool {
	return (this.property & (TERM_ANSI_JOIN | TERM_ANSI_NEST)) != 0
}

/*
Returns whether this keyspace is for a comma-separated join
*/
func (this *SubqueryTerm) IsCommaJoin() bool {
	return (this.property & TERM_COMMA_JOIN) != 0
}

/*
Returns whether it's lateral join
*/
func (this *SubqueryTerm) IsLateralJoin() bool {
	return (this.property & TERM_LATERAL_JOIN) != 0
}

/*
Set join hint
*/
func (this *SubqueryTerm) SetJoinHint(joinHint JoinHint) {
	this.joinHint = joinHint
}

/*
Set ANSI JOIN property
*/
func (this *SubqueryTerm) SetAnsiJoin() {
	this.property |= TERM_ANSI_JOIN
}

/*
Set ANSI NEST property
*/
func (this *SubqueryTerm) SetAnsiNest() {
	this.property |= TERM_ANSI_NEST
}

/*
Set COMMA JOIN property
*/
func (this *SubqueryTerm) SetCommaJoin() {
	this.property |= TERM_COMMA_JOIN
}

/*
Return whether correlated
*/
func (this *SubqueryTerm) IsCorrelated() bool {
	return this.subquery.IsCorrelated()
}

func (this *SubqueryTerm) GetCorrelation() map[string]uint32 {
	return this.correlation
}

/*
Unset (and save) join property
*/
func (this *SubqueryTerm) UnsetJoinProps() uint32 {
	joinProps := (this.property & TERM_JOIN_PROPS)
	this.property &^= TERM_JOIN_PROPS
	return joinProps
}

/*
Set join property
*/
func (this *SubqueryTerm) SetJoinProps(joinProps uint32) {
	this.property |= joinProps
}

func (this *SubqueryTerm) HasInferJoinHint() bool {
	return (this.property & TERM_INFER_JOIN_HINT) != 0
}

func (this *SubqueryTerm) SetInferJoinHint() {
	this.property |= TERM_INFER_JOIN_HINT
}

func (this *SubqueryTerm) HasTransferJoinHint() bool {
	return (this.property & TERM_XFER_JOIN_HINT) != 0
}

func (this *SubqueryTerm) SetTransferJoinHint() {
	this.property |= TERM_XFER_JOIN_HINT
}

func (this *SubqueryTerm) SetLateralJoin() {
	this.property |= TERM_LATERAL_JOIN
}

func (this *SubqueryTerm) UnsetLateralJoin() {
	this.property &^= TERM_LATERAL_JOIN
}

func (this *SubqueryTerm) SetErrorContext(line int, column int) {
	this.errorContext.Set(line, column)
}

func (this *SubqueryTerm) ErrorContext() string {
	return this.errorContext.String()
}
