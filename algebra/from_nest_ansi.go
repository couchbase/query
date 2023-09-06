//  Copyright 2017-Present Couchbase, Inc.
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
Nesting is conceptually the inverse of unnesting. Nesting performs a
join across two keyspaces (or a keyspace with itself). But instead of
producing a cross-product of the left and right hand inputs, a single
result is produced for each left hand input, while the corresponding
right hand inputs are collected into an array and nested as a single
array-valued field in the result object.
*/
type AnsiNest struct {
	left     FromTerm
	right    SimpleFromTerm
	outer    bool
	pushable bool // if not outer join, is the ON-clause pushable
	onclause expression.Expression
}

func NewAnsiNest(left FromTerm, outer bool, right SimpleFromTerm, onclause expression.Expression) *AnsiNest {
	return &AnsiNest{left, right, outer, false, onclause}
}

func (this *AnsiNest) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitAnsiNest(this)
}

/*
Maps the right input of the NEST if the left is mapped
successfully.
*/
func (this *AnsiNest) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	err = this.right.MapExpressions(mapper)
	if err != nil {
		return
	}

	this.onclause, err = mapper.Map(this.onclause)
	return
}

/*
Returns all contained Expressions.
*/
func (this *AnsiNest) Expressions() expression.Expressions {
	exprs := this.left.Expressions()
	exprs = append(exprs, this.right.Expressions()...)
	exprs = append(exprs, this.onclause)
	return exprs
}

/*
Returns all required privileges.
*/
func (this *AnsiNest) Privileges() (*auth.Privileges, errors.Error) {
	privs, err := this.left.Privileges()
	if err != nil {
		return nil, err
	}

	rprivs, err := this.right.Privileges()
	if err != nil {
		return nil, err
	}

	privs.AddAll(rprivs)
	privs.AddAll(this.onclause.Privileges())
	return privs, nil
}

/*
Representation as a N1QL string.
*/
func (this *AnsiNest) String() string {
	s := this.left.String()

	if this.outer {
		s += " left outer nest "
	} else {
		s += " nest "
	}

	s += this.right.String()
	s += " on "
	s += this.onclause.String()
	return s
}

/*
Qualify all identifiers for the parent expression. Checks if
a NEST alias exists and if it is a duplicate alias.
*/
func (this *AnsiNest) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	f, err = this.right.Formalize(f)
	if err != nil {
		return
	}

	this.onclause, err = f.Map(this.onclause)
	if err != nil {
		return
	}

	return
}

/*
Return the primary term in the left term of the NEST clause.
*/
func (this *AnsiNest) PrimaryTerm() SimpleFromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the nest alias of the right source.
*/
func (this *AnsiNest) Alias() string {
	return this.right.Alias()
}

/*
Returns the left term in the NEST clause.
*/
func (this *AnsiNest) Left() FromTerm {
	return this.left
}

/*
Returns the right term in the NEST clause.
*/
func (this *AnsiNest) Right() SimpleFromTerm {
	return this.right
}

/*
Returns boolean value based on if it is
an outer or inner NEST.
*/func (this *AnsiNest) Outer() bool {
	return this.outer
}

/*
Returns ON-clause of ANSI NEST
*/
func (this *AnsiNest) Onclause() expression.Expression {
	return this.onclause
}

/*
Set outer
*/
func (this *AnsiNest) SetOuter(outer bool) {
	this.outer = outer
}

/*
Set ON-clause
*/
func (this *AnsiNest) SetOnclause(onclause expression.Expression) {
	this.onclause = onclause
}

/*
Returns whether the ON-clause is pushable
*/
func (this *AnsiNest) Pushable() bool {
	return this.pushable
}

/*
Set pushable ON-clause
*/
func (this *AnsiNest) SetPushable(pushable bool) {
	this.pushable = pushable
}

/*
Returns whether contains correlation reference
*/
func (this *AnsiNest) IsCorrelated() bool {
	return joinCorrelated(this.left, this.right)
}

func (this *AnsiNest) GetCorrelation() map[string]uint32 {
	return getJoinCorrelation(this.left, this.right)
}

/*
Marshals input NEST terms into byte array.
*/
func (this *AnsiNest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "AnsiNest"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	r["onclause"] = this.onclause
	return json.Marshal(r)
}
