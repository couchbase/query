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
Represents the JOIN clause. Joins create new input objects by
combining two or more source objects.  They can be chained.
*/
type Join struct {
	left  FromTerm
	right *KeyspaceTerm
	outer bool
}

func NewJoin(left FromTerm, outer bool, right *KeyspaceTerm) *Join {
	return &Join{left, right, outer}
}

func (this *Join) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

/*
Maps left and right source objects of the JOIN.
*/
func (this *Join) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.right.MapExpressions(mapper)
}

/*
Returns all contained Expressions.
*/
func (this *Join) Expressions() expression.Expressions {
	return append(this.left.Expressions(), this.right.Expressions()...)
}

/*
Returns all required privileges.
*/
func (this *Join) Privileges() (*auth.Privileges, errors.Error) {
	privs, err := this.left.Privileges()
	if err != nil {
		return nil, err
	}

	rprivs, err := this.right.Privileges()
	if err != nil {
		return nil, err
	}

	privs.AddAll(rprivs)
	return privs, nil
}

/*
Representation as a N1QL string.
*/
func (this *Join) String() string {
	s := this.left.String()

	if this.outer {
		s += " left outer join "
	} else {
		s += " join "
	}

	s += this.right.String()
	return s
}

/*
Qualify all identifiers for the parent expression. Checks if
a JOIN alias exists and if it is a duplicate alias.
*/
func (this *Join) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	f.SetKeyspace("")
	this.right.joinKeys, err = f.Map(this.right.joinKeys)
	if err != nil {
		return
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewNoTermNameError("JOIN", "semantics.join.requires_name_or_alias")
		return nil, err
	}

	if ok := f.AllowedAlias(alias, true, false); ok {
		var errContext string
		if len(this.left.Expressions()) > 0 {
			errContext = this.left.Expressions()[0].ErrorContext()
		}
		err = errors.NewDuplicateAliasError("JOIN", alias+errContext, "semantics.join.duplicate_alias")
		return nil, err
	}

	f.SetAllowedAlias(alias, true)
	f.SetAlias(this.right.As())
	return
}

/*
Returns the primary term in the left source of
the JOIN.
*/
func (this *Join) PrimaryTerm() SimpleFromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the alias of the right source.
*/
func (this *Join) Alias() string {
	return this.right.Alias()
}

/*
Returns the left source object of the JOIN.
*/
func (this *Join) Left() FromTerm {
	return this.left
}

/*
Returns the right source object of the JOIN.
*/
func (this *Join) Right() *KeyspaceTerm {
	return this.right
}

/*
Returns boolean value based on if it is
an outer or inner JOIN.
*/
func (this *Join) Outer() bool {
	return this.outer
}

/*
Returns whether contains correlation reference
*/
func (this *Join) IsCorrelated() bool {
	return this.left.IsCorrelated()
}

func (this *Join) GetCorrelation() map[string]uint32 {
	return this.left.GetCorrelation()
}

/*
Marshals input JOIN terms.
*/
func (this *Join) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "join"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	return json.Marshal(r)
}
