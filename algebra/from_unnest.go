//  Copyright 2014-Present Couchbase, Inc.
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

/*
If a document or object contains a nested array, UNNEST conceptually
performs a join of the nested array with its parent object. Each
resulting joined object becomes an input to the query.
*/
type Unnest struct {
	left  FromTerm
	expr  expression.Expression
	as    string
	outer bool
}

func NewUnnest(left FromTerm, outer bool, expr expression.Expression, as string) *Unnest {
	return &Unnest{left, expr, as, outer}
}

func (this *Unnest) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

/*
Maps the source array of the unnest if the parent object(left)
is mapped successfully.
*/
func (this *Unnest) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	this.expr, err = mapper.Map(this.expr)
	return
}

/*
Returns all contained Expressions.
*/
func (this *Unnest) Expressions() expression.Expressions {
	return append(this.left.Expressions(), this.expr)
}

/*
Returns all required privileges.
*/
func (this *Unnest) Privileges() (*auth.Privileges, errors.Error) {
	lPrivs, err := this.left.Privileges()
	if err != nil {
		return lPrivs, err
	}
	lPrivs.AddAll(this.expr.Privileges())
	return lPrivs, nil
}

/*
Representation as a N1QL string.
*/
func (this *Unnest) String() string {
	s := this.left.String()

	if this.outer {
		s += " left outer unnest "
	} else {
		s += " unnest "
	}

	s += this.expr.String()

	if this.as != "" {
		s += " as `" + this.as + "`"
	}

	return s
}

/*
Qualify all identifiers for the parent expression. Checks if
a UNNEST alias exists and if it is a duplicate alias.
*/
func (this *Unnest) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	this.expr, err = f.Map(this.expr)
	if err != nil {
		return
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewNoTermNameError("UNNEST", "semantics.unnest.requires_name_or_alias")
		return nil, err
	}

	_, ok := f.Allowed().Field(alias)
	if ok {
		var errContext string
		if len(this.left.Expressions()) > 0 {
			errContext = this.left.Expressions()[0].ErrorContext()
		}
		err = errors.NewDuplicateAliasError("UNNEST", alias+errContext, "semantics.unnest.duplicate_alias")
		return nil, err
	}

	f.SetKeyspace("")
	f.SetAllowedUnnestAlias(alias)
	f.SetAlias(this.As())
	return
}

/*
Return the primary term in the parent object
(left term) of the UNNEST clause.
*/
func (this *Unnest) PrimaryTerm() SimpleFromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the UNNEST alias if set. Else returns the alias of
the input nested array.
*/
func (this *Unnest) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.expr.Alias()
	}
}

/*
Returns the left term (parent object) in the UNNEST
clause.
*/
func (this *Unnest) Left() FromTerm {
	return this.left
}

/*
Returns the source array object path expression for
the UNNEST clause.
*/
func (this *Unnest) Expression() expression.Expression {
	return this.expr
}

/*
Returns the alias string in an UNNEST clause.
*/
func (this *Unnest) As() string {
	return this.as
}

/*
Returns a boolean value depending on if it is
an outer or inner UNNEST.
*/
func (this *Unnest) Outer() bool {
	return this.outer
}

/*
Returns whether contains correlation reference
*/
func (this *Unnest) IsCorrelated() bool {
	return this.left.IsCorrelated()
}

func (this *Unnest) GetCorrelation() map[string]uint32 {
	return this.left.GetCorrelation()
}

/*
Marshals input unnest terms into byte array.
*/
func (this *Unnest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "unnest"}
	r["left"] = this.left
	r["expr"] = expression.NewStringer().Visit(this.expr)
	r["as"] = this.as
	r["outer"] = this.outer
	return json.Marshal(r)
}
