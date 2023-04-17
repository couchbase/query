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
Nesting is conceptually the inverse of unnesting. Nesting performs a
join across two keyspaces (or a keyspace with itself). But instead of
producing a cross-product of the left and right hand inputs, a single
result is produced for each left hand input, while the corresponding
right hand inputs are collected into an array and nested as a single
array-valued field in the result object.
*/
type Nest struct {
	left  FromTerm
	right *KeyspaceTerm
	outer bool
}

func NewNest(left FromTerm, outer bool, right *KeyspaceTerm) *Nest {
	return &Nest{left, right, outer}
}

func (this *Nest) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

/*
Maps the right input of the NEST if the left is mapped
successfully.
*/
func (this *Nest) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.right.MapExpressions(mapper)
}

/*
Returns all contained Expressions.
*/
func (this *Nest) Expressions() expression.Expressions {
	return append(this.left.Expressions(), this.right.Expressions()...)
}

/*
Returns all required privileges.
*/
func (this *Nest) Privileges() (*auth.Privileges, errors.Error) {
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
func (this *Nest) String() string {
	s := this.left.String()

	if this.outer {
		s += " left outer nest "
	} else {
		s += " nest "
	}

	s += this.right.String()
	return s
}

/*
Qualify all identifiers for the parent expression. Checks if
a NEST alias exists and if it is a duplicate alias.
*/
func (this *Nest) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
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
		err = errors.NewNoTermNameError("NEST", this.right.errorContext.String(), "semantics.nest.requires_name_or_alias")
		return nil, err
	}

	if ok := f.AllowedAlias(alias, true, false); ok {
		err = errors.NewDuplicateAliasError("NEST", alias, this.right.errorContext.String(), "semantics.nest.duplicate_alias")
		return nil, err
	}

	f.SetAllowedAlias(alias, true)
	f.SetAlias(this.right.As())
	return
}

/*
Return the primary term in the left term of the NEST clause.
*/
func (this *Nest) PrimaryTerm() SimpleFromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the NEST alias of the right source.
*/
func (this *Nest) Alias() string {
	return this.right.Alias()
}

/*
Returns the left term in the NEST clause.
*/
func (this *Nest) Left() FromTerm {
	return this.left
}

/*
Returns the right term in the NEST clause.
*/
func (this *Nest) Right() *KeyspaceTerm {
	return this.right
}

/*
Returns a boolean value depending on if it is
an outer or inner NEST.
*/
func (this *Nest) Outer() bool {
	return this.outer
}

/*
Returns whether contains correlation reference
*/
func (this *Nest) IsCorrelated() bool {
	return this.left.IsCorrelated()
}

func (this *Nest) GetCorrelation() map[string]uint32 {
	return this.left.GetCorrelation()
}

/*
Marshals input NEST terms into byte array.
*/
func (this *Nest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "nest"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	return json.Marshal(r)
}
