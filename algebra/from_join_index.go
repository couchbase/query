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
Represents the JOIN clause. IndexJoins create new input objects by
combining two or more source objects.  They can be chained.
*/
type IndexJoin struct {
	left   FromTerm
	right  *KeyspaceTerm
	outer  bool
	keyFor string
}

func NewIndexJoin(left FromTerm, outer bool, right *KeyspaceTerm, keyFor string) *IndexJoin {
	return &IndexJoin{left, right, outer, keyFor}
}

func (this *IndexJoin) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitIndexJoin(this)
}

/*
Maps left and right source objects of the JOIN.
*/
func (this *IndexJoin) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.right.MapExpressions(mapper)
}

/*
   Returns all contained Expressions.
*/
func (this *IndexJoin) Expressions() expression.Expressions {
	return append(this.left.Expressions(), this.right.Expressions()...)
}

/*
Returns all required privileges.
*/
func (this *IndexJoin) Privileges() (*auth.Privileges, errors.Error) {
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
func (this *IndexJoin) String() string {
	s := this.left.String()

	if this.outer {
		s += " left outer join "
	} else {
		s += " join "
	}

	s += this.right.String()
	s += " for " + this.keyFor
	return s
}

/*
Qualify all identifiers for the parent expression. Checks if
a JOIN alias exists and if it is a duplicate alias.
*/
func (this *IndexJoin) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	_, ok := f.Allowed().Field(this.keyFor)
	if !ok {
		err = errors.NewUnknownForError("JOIN", this.keyFor, "semantics.join.unknown_for")
		return nil, err
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewNoTermNameError("JOIN", "semantics.join.requires_name_or_alias")
		return nil, err
	}

	_, ok = f.Allowed().Field(alias)
	if ok {
		err = errors.NewDuplicateAliasError("JOIN", alias, "semantics.join.duplicate_alias")
		return nil, err
	}

	f.SetAllowedAlias(alias, true)
	f.SetKeyspace("")

	p := expression.NewFormalizer("", parent)
	p.SetAllowedAlias(alias, true)
	this.right.joinKeys, err = p.Map(this.right.joinKeys)

	for ident, val := range p.Identifiers().Fields() {
		f.Identifiers().SetField(ident, val)
	}

	f.SetAlias(this.right.As())

	return
}

/*
Returns the primary term in the left source of
the JOIN.
*/
func (this *IndexJoin) PrimaryTerm() SimpleFromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the alias of the right source.
*/
func (this *IndexJoin) Alias() string {
	return this.right.Alias()
}

/*
Returns the left source object of the JOIN.
*/
func (this *IndexJoin) Left() FromTerm {
	return this.left
}

/*
Returns the right source object of the JOIN.
*/
func (this *IndexJoin) Right() *KeyspaceTerm {
	return this.right
}

func (this *IndexJoin) Outer() bool {
	return this.outer
}

func (this *IndexJoin) For() string {
	return this.keyFor
}

/*
Returns whether contains correlation reference
*/
func (this *IndexJoin) IsCorrelated() bool {
	return this.left.IsCorrelated() || this.right.IsCorrelated()
}

/*
Marshals input JOIN terms.
*/
func (this *IndexJoin) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "indexJoin"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	r["for"] = this.keyFor
	return json.Marshal(r)
}
