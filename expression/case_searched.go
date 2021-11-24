//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/value"
)

/*
Searched case expressions allow for conditional logic within
an expression. It contains When and Else terms.
*/
type SearchedCase struct {
	ExpressionBase
	whenTerms WhenTerms
	elseTerm  Expression
}

func NewSearchedCase(whenTerms WhenTerms, elseTerm Expression) Expression {
	rv := &SearchedCase{
		whenTerms: whenTerms,
		elseTerm:  elseTerm,
	}

	rv.setConditional()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *SearchedCase) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSearchedCase(this)
}

/*
If every term returns the same type or UNKNOWN, return that
type. Else, return value.JSON.
*/
func (this *SearchedCase) Type() value.Type {
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
The first WHEN expression is evaluated. If TRUE, the result of this
expression is the THEN expression. If not, subsequent WHEN clauses
are evaluated in the same manner. If none of the WHEN clauses
evaluate to TRUE, then the result of the expression is the ELSE
expression. If no ELSE expression was provided, the result is NULL.
*/
func (this *SearchedCase) Evaluate(item value.Value, context Context) (value.Value, error) {
	for _, w := range this.whenTerms {
		wv, err := w.When.Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		if wv.Truth() {
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

func (this *SearchedCase) DependsOn(other Expression) bool {
	return this.dependsOn(other)
}

func (this *SearchedCase) Children() Expressions {
	rv := make(Expressions, 0, 1+(len(this.whenTerms)<<1))

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
This method maps the when and else terms to an expression. If
there is an error during the mapping it is returned. Range over
the when terms and map them along with the then terms. If an
error is encountered at either mapping return it. If an else
term is pressent map it. Return the error encountered.
*/
func (this *SearchedCase) MapChildren(mapper Mapper) (err error) {
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

func (this *SearchedCase) Copy() Expression {
	rv := NewSearchedCase(this.whenTerms.Copy(), Copy(this.elseTerm))
	rv.BaseCopy(this)
	return rv
}
