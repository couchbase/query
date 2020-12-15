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
	"github.com/couchbase/query/value"
)

/*
SelectTerm wraps a Select and implements the Node interface. It
enables Select to be a term in a UNION, INTERSECT, or EXCEPT.
*/
type SelectTerm struct {
	query *Select `json:"query"`
}

/*
Constructor.
*/
func NewSelectTerm(term *Select) *SelectTerm {
	return &SelectTerm{term}
}

/*
Visitor pattern.
*/
func (this *SelectTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitSelectTerm(this)
}

/*
This method returns the shape of the select clause. It returns a value
that represents the signature of the projection.
*/
func (this *SelectTerm) Signature() value.Value {
	return this.query.Signature()
}

/*
 */
func (this *SelectTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	err = this.query.FormalizeSubquery(parent)
	if err != nil {
		return nil, err
	}

	return expression.NewFormalizer("", nil), nil
}

/*
This method maps all the constituent clauses, namely the from,
let, where, group by and projection(select) within a SelectTerm
statement.
*/
func (this *SelectTerm) MapExpressions(mapper expression.Mapper) (err error) {
	return this.query.MapExpressions(mapper)
}

/*
   Returns all contained Expressions.
*/
func (this *SelectTerm) Expressions() expression.Expressions {
	return this.query.Expressions()
}

/*
   Result terms.
*/
func (this *SelectTerm) ResultTerms() ResultTerms {
	return this.query.Subresult().ResultTerms()
}

/*
   Raw projection.
*/
func (this *SelectTerm) Raw() bool {
	return this.query.Subresult().Raw()
}

/*
Returns all required privileges.
*/
func (this *SelectTerm) Privileges() (*auth.Privileges, errors.Error) {
	return this.query.Privileges()
}

/*
   Representation as a N1QL string.
*/
func (this *SelectTerm) String() string {
	return "(" + this.query.String() + ")"
}

/*
Returns bool value that depicts if query is correlated
or not.
*/
func (this *SelectTerm) IsCorrelated() bool {
	return this.query.IsCorrelated()
}

/*
Returns estimated result size
*/
func (this *SelectTerm) EstResultSize() int64 {
	return this.query.EstResultSize()
}

/*
Accessor.
*/
func (this *SelectTerm) Select() *Select {
	return this.query
}
