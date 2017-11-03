//  Copyright (c) 2017 Couchbase, Inc.
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
	right    FromTerm
	outer    bool
	onclause expression.Expression
}

func NewAnsiJoin(left FromTerm, outer bool, right FromTerm, onclause expression.Expression) *AnsiJoin {
	return &AnsiJoin{left, right, outer, onclause}
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

	this.onclause, err = mapper.Map(this.onclause)
	return
}

/*
   Returns all contained Expressions.
*/
func (this *AnsiJoin) Expressions() expression.Expressions {
	exprs := this.left.Expressions()
	exprs = append(exprs, this.right.Expressions()...)
	exprs = append(exprs, this.onclause)
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
	privs.AddAll(this.onclause.Privileges())
	return privs, nil
}

/*
   Representation as a N1QL string.
*/
func (this *AnsiJoin) String() string {
	s := this.left.String()

	if this.outer {
		s += " left outer join "
	} else {
		s += " join "
	}

	s += this.right.String()
	s += " on "
	s += this.onclause.String()
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

	this.onclause, err = f.Map(this.onclause)
	if err != nil {
		return
	}

	return
}

/*
Returns the primary term in the left source of
the JOIN.
*/
func (this *AnsiJoin) PrimaryTerm() FromTerm {
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
func (this *AnsiJoin) Right() FromTerm {
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
Marshals input JOIN terms.
*/
func (this *AnsiJoin) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "AnsiJoin"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	r["onclause"] = this.onclause
	return json.Marshal(r)
}
