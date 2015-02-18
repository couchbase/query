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
	"github.com/couchbase/query/expression"
)

/*
This represents the Order by clause. Type Order is a
struct that contains the ordering terms called sort
terms.
*/
type Order struct {
	terms SortTerms
}

/*
The function NewOrder returns a pointer to the Order
struct that has its field sort terms set to the input
argument terms.
*/
func NewOrder(terms SortTerms) *Order {
	return &Order{
		terms: terms,
	}
}

/*
Map expressions for the terms by calling MapExpressions.
*/
func (this *Order) MapExpressions(mapper expression.Mapper) error {
	return this.terms.MapExpressions(mapper)
}

/*
   Returns all contained Expressions.
*/
func (this *Order) Expressions() expression.Expressions {
	return this.terms.Expressions()
}

/*
   Representation as a N1QL string.
*/
func (this *Order) String() string {
	return " order by " + this.terms.String()
}

/*
Return the ordering terms (sort terms).
*/
func (this *Order) Terms() SortTerms {
	return this.terms
}

/*
It represents multiple orderby terms.
Type SortTerms is a slice of SortTerm.
*/
type SortTerms []*SortTerm

/*
Represents the ordering term in an order by clause. Type
SortTerm is a struct containing the expression and a bool
value that decides the sort order (ASC or DESC).
*/
type SortTerm struct {
	expr       expression.Expression `json:"expr"`
	descending bool                  `json:"desc"`
}

/*
The function NewSortTerm returns a pointer to the SortTerm
struct that has its fields set to the input arguments.
*/
func NewSortTerm(expr expression.Expression, descending bool) *SortTerm {
	return &SortTerm{
		expr:       expr,
		descending: descending,
	}
}

/*
   Representation as a N1QL string.
*/
func (this *SortTerm) String() string {
	s := this.expr.String()

	if this.descending {
		s += " desc"
	}

	return s
}

/*
Return the expression that is sorted in the order
by clause.
*/
func (this *SortTerm) Expression() expression.Expression {
	return this.expr
}

/*
Return bool value representing ASC or DESC sort order.
*/
func (this *SortTerm) Descending() bool {
	return this.descending
}

/*
Map Expressions for all sort terms in the receiver.
*/
func (this SortTerms) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this {
		term.expr, err = mapper.Map(term.expr)
		if err != nil {
			return
		}
	}

	return
}

/*
   Returns all contained Expressions.
*/
func (this SortTerms) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, len(this))

	for i, term := range this {
		exprs[i] = term.expr
	}

	return exprs
}

/*
   Representation as a N1QL string.
*/
func (this SortTerms) String() string {
	s := ""

	for i, term := range this {
		if i > 0 {
			s += ", "
		}

		s += term.String()
	}

	return s
}
