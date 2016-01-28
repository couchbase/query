//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

/*
Represents the join clause. IndexJoins create new input objects by
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
Maps left and right source objects of the join.
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
func (this *IndexJoin) Privileges() (datastore.Privileges, errors.Error) {
	privs, err := this.left.Privileges()
	if err != nil {
		return nil, err
	}

	rprivs, err := this.right.Privileges()
	if err != nil {
		return nil, err
	}

	privs.Add(rprivs)
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

	s += this.right.toString(true)
	return s
}

/*
Qualify all identifiers for the parent expression. Checks is
a join alias exists and if it is a duplicate alias.
*/
func (this *IndexJoin) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	_, ok := f.Allowed().Field(this.keyFor)
	if !ok {
		err = errors.NewUnknownForError("JOIN", this.keyFor, "plan.join.unknown_for")
		return nil, err
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewNoTermNameError("JOIN", "plan.join.requires_name_or_alias")
		return nil, err
	}

	_, ok = f.Allowed().Field(alias)
	if ok {
		err = errors.NewDuplicateAliasError("JOIN", alias, "plan.join.duplicate_alias")
		return nil, err
	}

	f.Allowed().SetField(alias, alias)
	f.SetKeyspace("")

	p := expression.NewFormalizer("", parent)
	p.Allowed().SetField(alias, alias)
	this.right.keys, err = p.Map(this.right.keys)

	for ident, val := range p.Identifiers() {
		f.Identifiers()[ident] = val
	}

	return
}

/*
Returns the primary term in the left source of
the join.
*/
func (this *IndexJoin) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the alias of the right source.
*/
func (this *IndexJoin) Alias() string {
	return this.right.Alias()
}

/*
Returns the left source object of the join.
*/
func (this *IndexJoin) Left() FromTerm {
	return this.left
}

/*
Returns the right source object of the join.
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
Marshals input join terms.
*/
func (this *IndexJoin) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "indexJoin"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	r["for"] = this.keyFor
	return json.Marshal(r)
}
