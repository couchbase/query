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
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

type SubqueryTerm struct {
	subquery *Select
	as       string
}

/*
Constructor.
*/
func NewSubqueryTerm(subquery *Select, as string) *SubqueryTerm {
	return &SubqueryTerm{subquery, as}
}

/*
Visitor pattern.
*/
func (this *SubqueryTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitSubqueryTerm(this)
}

/*
Apply mapping to all contained Expressions.
*/
func (this *SubqueryTerm) MapExpressions(mapper expression.Mapper) (err error) {
	return this.subquery.MapExpressions(mapper)
}

/*
   Returns all contained Expressions.
*/
func (this *SubqueryTerm) Expressions() expression.Expressions {
	return this.subquery.Expressions()
}

/*
Returns all required privileges.
*/
func (this *SubqueryTerm) Privileges() (*auth.Privileges, errors.Error) {
	return this.subquery.Privileges()
}

/*
   Representation as a N1QL string.
*/
func (this *SubqueryTerm) String() string {
	return "(" + this.subquery.String() + ") as " + this.as
}

/*
Qualify all identifiers for the parent expression. Checks for
duplicate aliases.
*/
func (this *SubqueryTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	err = this.subquery.Formalize()
	if err != nil {
		return
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewNoTermNameError("FROM Subquery", "plan.subquery.requires_name_or_alias")
		return
	}

	_, ok := parent.Allowed().Field(alias)
	if ok {
		err = errors.NewDuplicateAliasError("subquery", alias, "plan.subquery.duplicate_alias")
		return nil, err
	}

	f = expression.NewFormalizer(alias, parent)
	f.SetAlias(this.Alias())
	return
}

/*
Return the primary term in the from clause.
*/
func (this *SubqueryTerm) PrimaryTerm() FromTerm {
	return this
}

/*
Returns the Alias string.
*/
func (this *SubqueryTerm) Alias() string {
	return this.as
}

/*
Returns the inner subquery.
*/
func (this *SubqueryTerm) Subquery() *Select {
	return this.subquery
}
