//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

type ExpressionTerm struct {
	fromExpr     expression.Expression
	as           string
	keyspaceTerm *KeyspaceTerm
	isKeyspace   bool
	correlated   bool
	joinHint     JoinHint
	property     uint32
	correlation  map[string]uint32
}

/*
Constructor.
*/
func NewExpressionTerm(fromExpr expression.Expression, as string,
	keyspaceTerm *KeyspaceTerm, isKeyspace bool, joinHint JoinHint) *ExpressionTerm {
	return &ExpressionTerm{fromExpr, as, keyspaceTerm, isKeyspace, false, joinHint, 0, nil}
}

/*
Visitor pattern.
*/
func (this *ExpressionTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitExpressionTerm(this)
}

/*
Apply mapping to all contained Expressions.
*/
func (this *ExpressionTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if this.isKeyspace {
		return this.keyspaceTerm.MapExpressions(mapper)
	} else {
		this.fromExpr, err = mapper.Map(this.fromExpr)
	}
	return err
}

/*
Returns all contained Expressions.
*/
func (this *ExpressionTerm) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 2)

	if this.isKeyspace {
		exprs = append(exprs, this.keyspaceTerm.Expressions()...)
	} else {
		exprs = append(exprs, this.fromExpr)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *ExpressionTerm) Privileges() (*auth.Privileges, errors.Error) {
	if this.isKeyspace {
		return this.keyspaceTerm.Privileges()
	}
	return this.fromExpr.Privileges(), nil
}

/*
Representation as a N1QL string.
*/
func (this *ExpressionTerm) String() string {
	s := ""
	if this.isKeyspace {
		s = this.keyspaceTerm.String()
	} else {
		s = this.fromExpr.String()
		if _, ok := this.fromExpr.(*expression.Identifier); ok {
			s = "(" + s + ")"
		}

		if this.as != "" {
			s += " as `" + this.as + "`"
		}
		if jhs := this.joinHint.String(); len(jhs) > 0 {
			s += " use " + jhs
		}
	}
	return s
}

/*
Qualify all identifiers for the parent expression. Checks for
duplicate aliases.
*/
func (this *ExpressionTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	if this.keyspaceTerm != nil {
		path := this.keyspaceTerm.Path()
		_, isIdentifier := this.fromExpr.(*expression.Identifier)

		// MB-46856 if the expression path is longer than 1, use the bucket
		if !isIdentifier && path.IsCollection() {
			_, ok := parent.Aliases().Field(path.Bucket())
			this.isKeyspace = !ok
		} else {
			_, ok := parent.Aliases().Field(this.keyspaceTerm.Keyspace())
			this.isKeyspace = !ok
		}
	}

	if this.isKeyspace {
		this.keyspaceTerm.SetProperty(this.property)
		this.keyspaceTerm.SetJoinHint(this.joinHint)
		return this.keyspaceTerm.Formalize(parent)
	}

	alias := this.Alias()
	if alias == "" {
		var errContext string
		if this.fromExpr != nil {
			errContext = this.fromExpr.ErrorContext()
		}
		err = errors.NewNoTermNameError("FROM expression"+errContext, "semantics.fromExpr.requires_name_or_alias")
		return nil, err
	}

	_, ok := parent.Allowed().Field(alias)
	if ok && !parent.WithAlias(alias) {
		var errContext string
		if this.fromExpr != nil {
			errContext = this.fromExpr.ErrorContext()
		}
		err = errors.NewDuplicateAliasError("FROM expression"+errContext, alias, "semantics.fromExpr.duplicate_alias")
		return nil, err
	}

	if this.keyspaceTerm != nil && (this.keyspaceTerm.Keys() != nil || this.keyspaceTerm.Indexes() != nil) {
		err = errors.NewUseKeysUseIndexesError("FROM expression", "semantics.fromExpr.no_usekeys_or_useindex")
		return nil, err
	}

	f1 := expression.NewFormalizer("", parent)
	this.fromExpr, err = f1.Map(this.fromExpr)
	if err != nil {
		return
	}

	// Determine if this expression contains any correlated references
	this.correlated = f1.CheckCorrelated()
	if this.correlated {
		this.correlation = addSimpleTermCorrelation(this.correlation, f1.GetCorrelation(),
			this.IsAnsiJoinOp(), parent)
	}

	// for checking fromExpr we need a new formalizer, however, if this ExpressionTerm
	// is under an ANSI join/nest operation we need to use the parent's formalizer
	if this.IsAnsiJoinOp() {
		f = parent
		f.SetKeyspace("")
	} else {
		f = f1
		f.SetExprSubqKeyspace(alias)
	}
	f.SetAllowedExprTermAlias(alias)
	f.SetAlias(this.as)
	return
}

/*
Return the primary term in the from clause.
*/
func (this *ExpressionTerm) PrimaryTerm() SimpleFromTerm {
	return this
}

/*
Returns the Alias string.
*/
func (this *ExpressionTerm) Alias() string {
	if this.isKeyspace {
		return this.keyspaceTerm.Alias()
	} else if this.as != "" {
		return this.as
	} else {
		return this.fromExpr.Alias()
	}
}

/*
Returns the from Expression
*/
func (this *ExpressionTerm) ExpressionTerm() expression.Expression {
	return this.fromExpr
}

/*
Returns the Keyspace Term
*/
func (this *ExpressionTerm) KeyspaceTerm() *KeyspaceTerm {
	return this.keyspaceTerm
}

/*
Returns the if Expression is Keyspace
*/
func (this *ExpressionTerm) IsKeyspace() bool {
	return this.isKeyspace
}

/*
Returns if Expression is (lateral) correlated
i.e., refers to any keyspace before the expression term in FROM clause
*/
func (this *ExpressionTerm) IsCorrelated() bool {
	if this.isKeyspace {
		return this.keyspaceTerm.IsCorrelated()
	}
	return this.correlated
}

func (this *ExpressionTerm) GetCorrelation() map[string]uint32 {
	if this.isKeyspace {
		return this.keyspaceTerm.GetCorrelation()
	}
	return this.correlation
}

/*
Returns the join hint
*/
func (this *ExpressionTerm) JoinHint() JoinHint {
	return this.joinHint
}

/*
Join hint prefers hash join
*/
func (this *ExpressionTerm) PreferHash() bool {
	return this.joinHint == USE_HASH_BUILD || this.joinHint == USE_HASH_PROBE || this.joinHint == USE_HASH_EITHER
}

/*
Join hint prefers nested loop join
*/
func (this *ExpressionTerm) PreferNL() bool {
	return this.joinHint == USE_NL
}

/*
Returns the property.
*/
func (this *ExpressionTerm) Property() uint32 {
	return this.property
}

/*
Returns whether this expression term is for an ANSI JOIN
*/
func (this *ExpressionTerm) IsAnsiJoin() bool {
	return (this.property & TERM_ANSI_JOIN) != 0
}

/*
Returns whether this expression term is for an ANSI NEST
*/
func (this *ExpressionTerm) IsAnsiNest() bool {
	return (this.property & TERM_ANSI_NEST) != 0
}

/*
Returns whether this expression term is for an ANSI JOIN or ANSI NEST
*/
func (this *ExpressionTerm) IsAnsiJoinOp() bool {
	return (this.property & (TERM_ANSI_JOIN | TERM_ANSI_NEST)) != 0
}

/*
Returns whether this keyspace is for a comma-separated join
*/
func (this *ExpressionTerm) IsCommaJoin() bool {
	return (this.property & TERM_COMMA_JOIN) != 0
}

/*
Set the from Expression
*/
func (this *ExpressionTerm) SetExpressionTerm(fromExpr expression.Expression) {
	this.fromExpr = fromExpr
}

/*
Set join hint
*/
func (this *ExpressionTerm) SetJoinHint(joinHint JoinHint) {
	this.joinHint = joinHint
}

/*
Set ANSI JOIN property
*/
func (this *ExpressionTerm) SetAnsiJoin() {
	this.property |= TERM_ANSI_JOIN
}

/*
Set ANSI NEST property
*/
func (this *ExpressionTerm) SetAnsiNest() {
	this.property |= TERM_ANSI_NEST
}

/*
Set COMMA JOIN property
*/
func (this *ExpressionTerm) SetCommaJoin() {
	this.property |= TERM_COMMA_JOIN
}

/*
Unset (and save) join property
*/
func (this *ExpressionTerm) UnsetJoinProps() uint32 {
	joinProps := (this.property & TERM_JOIN_PROPS)
	this.property &^= TERM_JOIN_PROPS
	return joinProps
}

/*
Set join property
*/
func (this *ExpressionTerm) SetJoinProps(joinProps uint32) {
	this.property |= joinProps
}

/*
Marshals input ExpressionTerm.
*/
func (this *ExpressionTerm) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "ExpressionTerm"}
	r["as"] = this.as
	r["fromexpr"] = this.fromExpr
	if this.correlated {
		r["correlated"] = this.correlated
	}
	return json.Marshal(r)
}
