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
type IndexNest struct {
	left   FromTerm
	right  *KeyspaceTerm
	outer  bool
	keyFor string
}

func NewIndexNest(left FromTerm, outer bool, right *KeyspaceTerm, keyFor string) *IndexNest {
	return &IndexNest{left, right, outer, keyFor}
}

func (this *IndexNest) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitIndexNest(this)
}

/*
Maps the right input of the NEST if the left is mapped
successfully.
*/
func (this *IndexNest) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.right.MapExpressions(mapper)
}

/*
Returns all contained Expressions.
*/
func (this *IndexNest) Expressions() expression.Expressions {
	return append(this.left.Expressions(), this.right.Expressions()...)
}

/*
Returns all required privileges.
*/
func (this *IndexNest) Privileges() (*auth.Privileges, errors.Error) {
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
func (this *IndexNest) String() string {
	s := this.left.String()

	if this.outer {
		s += " left outer nest "
	} else {
		s += " nest "
	}

	s += this.right.String()
	s += " for " + this.keyFor
	return s
}

/*
Qualify all identifiers for the parent expression. Checks if
a NEST alias exists and if it is a duplicate alias.
*/
func (this *IndexNest) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	_, ok := f.Allowed().Field(this.keyFor)
	if !ok {
		err = errors.NewUnknownForError("NEST", this.keyFor, "semantics.nest.unknown_for")
		return nil, err
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewNoTermNameError("NEST", "semantics.nest.requires_name_or_alias")
		return nil, err
	}

	_, ok = f.Allowed().Field(alias)
	if ok {
		var errContext string
		if len(this.left.Expressions()) > 0 {
			errContext = this.left.Expressions()[0].ErrorContext()
		}
		err = errors.NewDuplicateAliasError("NEST", alias+errContext, "semantics.nest.duplicate_alias")
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
Return the primary term in the left term of the NEST clause.
*/
func (this *IndexNest) PrimaryTerm() SimpleFromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the nest alias of the right source.
*/
func (this *IndexNest) Alias() string {
	return this.right.Alias()
}

/*
Returns the left term in the NEST clause.
*/
func (this *IndexNest) Left() FromTerm {
	return this.left
}

/*
Returns the right term in the NEST clause.
*/
func (this *IndexNest) Right() *KeyspaceTerm {
	return this.right
}

func (this *IndexNest) Outer() bool {
	return this.outer
}

func (this *IndexNest) For() string {
	return this.keyFor
}

/*
Returns whether contains correlation reference
*/
func (this *IndexNest) IsCorrelated() bool {
	return this.left.IsCorrelated()
}

func (this *IndexNest) GetCorrelation() map[string]uint32 {
	return this.left.GetCorrelation()
}

/*
Marshals input NEST terms into byte array.
*/
func (this *IndexNest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "indexNest"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	r["for"] = this.keyFor
	return json.Marshal(r)
}
