//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbase/query/value"
)

/*
Simple case expressions allow for conditional matching within an expression.
It contains search expression, when/then terms and an optional else
expression.
*/
type SimpleCase struct {
	ExpressionBase
	searchTerm Expression
	whenTerms  WhenTerms
	elseTerm   Expression
}

func NewSimpleCase(searchTerm Expression, whenTerms WhenTerms, elseTerm Expression) Expression {
	rv := &SimpleCase{
		searchTerm: searchTerm,
		whenTerms:  whenTerms,
		elseTerm:   elseTerm,
	}

	rv.conditional = true
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *SimpleCase) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSimpleCase(this)
}

/*
If every term returns the same type or UNKNOWN, return that
type. Else, return value.JSON.
*/
func (this *SimpleCase) Type() value.Type {
	t := value.NULL
	if this.elseTerm != nil {
		t = this.elseTerm.Type()
	}

	for _, w := range this.whenTerms {
		tt := w.Then.Type()
		if t > value.NULL && tt > value.NULL && tt != t {
			return value.JSON
		} else if t < tt {
			t = tt
		}
	}

	return t
}

/*
The first WHEN expression is evaluated. If it is equal to the
search expression, the result of this expression is the THEN
expression. If not, subsequent WHEN clauses are evaluated in
the same manner. If none of the WHEN expressions is equal to
the search expression, then the result of the CASE expression
is the ELSE expression. If no ELSE expression was provided,
the result is NULL.
*/
func (this *SimpleCase) Evaluate(item value.Value, context Context) (value.Value, error) {
	s, err := this.searchTerm.Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	for _, w := range this.whenTerms {
		wv, err := w.When.Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		if s.Equals(wv).Truth() {
			tv, err := w.Then.Evaluate(item, context)
			if err != nil {
				return nil, err
			}

			return tv, nil
		}
	}

	if this.elseTerm == nil {
		return value.NULL_VALUE, nil
	}

	ev, err := this.elseTerm.Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	return ev, nil
}

func (this *SimpleCase) DependsOn(other Expression) bool {
	return this.searchTerm.DependsOn(other) &&
		(this.elseTerm == nil || this.elseTerm.DependsOn(other))
}

func (this *SimpleCase) Children() Expressions {
	rv := make(Expressions, 0, 2+(len(this.whenTerms)<<1))
	rv = append(rv, this.searchTerm)

	for _, w := range this.whenTerms {
		rv = append(rv, w.When)
		rv = append(rv, w.Then)
	}

	if this.elseTerm != nil {
		rv = append(rv, this.elseTerm)
	}

	return rv
}

/*
This method maps the search, when and else terms to an expression.
If there is an error during the mapping it is returned. Map the
search term first. Range over the when terms and map them along
with the then terms. If an error is encountered at either mapping
return it. If an else term is pressent map it. Return the error
encountered.
*/
func (this *SimpleCase) MapChildren(mapper Mapper) (err error) {
	this.searchTerm, err = mapper.Map(this.searchTerm)
	if err != nil {
		return
	}

	for _, w := range this.whenTerms {
		w.When, err = mapper.Map(w.When)
		if err != nil {
			return
		}

		w.Then, err = mapper.Map(w.Then)
		if err != nil {
			return
		}
	}

	if this.elseTerm != nil {
		this.elseTerm, err = mapper.Map(this.elseTerm)
		if err != nil {
			return
		}
	}

	return
}

func (this *SimpleCase) Copy() Expression {
	return NewSimpleCase(this.searchTerm.Copy(), this.whenTerms.Copy(), Copy(this.elseTerm))
}
