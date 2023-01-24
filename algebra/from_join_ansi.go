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
Represents the ANSI JOIN clause. AnsiJoins create new input objects by
combining two or more source objects.  They can be chained.
*/
type AnsiJoin struct {
	left     FromTerm
	right    SimpleFromTerm
	outer    bool
	pushable bool // if not outer join, is the ON-clause pushable
	onclause expression.Expression
}

func NewAnsiJoin(left FromTerm, outer bool, right SimpleFromTerm, onclause expression.Expression) *AnsiJoin {
	return &AnsiJoin{left, right, outer, false, onclause}
}

func NewAnsiRightJoin(left SimpleFromTerm, right SimpleFromTerm, onclause expression.Expression) *AnsiJoin {
	TransferJoinHint(left, right)
	return &AnsiJoin{right, left, true, false, onclause}
}

func TransferJoinHint(left SimpleFromTerm, right SimpleFromTerm) {
	// transfer join hint from right-hand side to left-hand side
	joinHint := right.JoinHint()
	if joinHint != JOIN_HINT_NONE {
		// swith build and probe side of USE HASH hint
		switch joinHint {
		case USE_HASH_BUILD:
			joinHint = USE_HASH_PROBE
		case USE_HASH_PROBE:
			joinHint = USE_HASH_BUILD
		}
		left.SetJoinHint(joinHint)
		right.SetJoinHint(JOIN_HINT_NONE)
	}
}

func (this *AnsiJoin) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitAnsiJoin(this)
}

/*
Maps left and right source objects of the JOIN.
*/
func (this *AnsiJoin) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	err = this.right.MapExpressions(mapper)
	if err != nil {
		return
	}

	if !this.IsCommaJoin() {
		this.onclause, err = mapper.Map(this.onclause)
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *AnsiJoin) Expressions() expression.Expressions {
	exprs := this.left.Expressions()
	exprs = append(exprs, this.right.Expressions()...)
	if !this.IsCommaJoin() {
		exprs = append(exprs, this.onclause)
	}
	return exprs
}

/*
Returns all required privileges.
*/
func (this *AnsiJoin) Privileges() (*auth.Privileges, errors.Error) {
	privs, err := this.left.Privileges()
	if err != nil {
		return nil, err
	}

	rprivs, err := this.right.Privileges()
	if err != nil {
		return nil, err
	}

	privs.AddAll(rprivs)
	if !this.IsCommaJoin() {
		privs.AddAll(this.onclause.Privileges())
	}

	return privs, nil
}

/*
Representation as a N1QL string.
*/
func (this *AnsiJoin) String() string {
	s := this.left.String()

	commaJoin := this.IsCommaJoin()
	if commaJoin {
		s += ", "
	} else if this.outer {
		s += " left outer join "
	} else {
		s += " join "
	}

	s += this.right.String()
	if !commaJoin {
		s += " on "
		s += this.onclause.String()
	}

	return s
}

/*
Qualify all identifiers for the parent expression. Checks if
a JOIN alias exists and if it is a duplicate alias.
*/
func (this *AnsiJoin) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	f, err = this.right.Formalize(f)
	if err != nil {
		return
	}

	if !this.IsCommaJoin() {
		this.onclause, err = f.Map(this.onclause)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns the primary term in the left source of
the JOIN.
*/
func (this *AnsiJoin) PrimaryTerm() SimpleFromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the alias of the right source.
*/
func (this *AnsiJoin) Alias() string {
	return this.right.Alias()
}

/*
Returns the left source object of the JOIN.
*/
func (this *AnsiJoin) Left() FromTerm {
	return this.left
}

/*
Returns the right source object of the JOIN.
*/
func (this *AnsiJoin) Right() SimpleFromTerm {
	return this.right
}

/*
Returns boolean value based on if it is
an outer or inner JOIN.
*/
func (this *AnsiJoin) Outer() bool {
	return this.outer
}

/*
Returns ON-clause of ANSI JOIN
*/
func (this *AnsiJoin) Onclause() expression.Expression {
	return this.onclause
}

/*
Set outer
*/
func (this *AnsiJoin) SetOuter(outer bool) {
	this.outer = outer
}

/*
Set ON-clause
*/
func (this *AnsiJoin) SetOnclause(onclause expression.Expression) {
	this.onclause = onclause
}

/*
Returns whether the ON-clause is pushable
*/
func (this *AnsiJoin) Pushable() bool {
	return this.pushable
}

/*
Set pushable ON-clause
*/
func (this *AnsiJoin) SetPushable(pushable bool) {
	this.pushable = pushable
}

/*
Returns whether contains correlation reference
*/
func (this *AnsiJoin) IsCorrelated() bool {
	return this.left.IsCorrelated() || this.right.IsCorrelated()
}

/*
Returns whether this is a comma-separated join
*/
func (this *AnsiJoin) IsCommaJoin() bool {
	return this.right.IsCommaJoin()
}

/*
Marshals input JOIN terms.
*/
func (this *AnsiJoin) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "AnsiJoin"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	if !this.IsCommaJoin() {
		r["onclause"] = this.onclause
	}
	return json.Marshal(r)
}
