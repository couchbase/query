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
	"github.com/couchbaselabs/query/value"
)

type SearchedCase struct {
	ExpressionBase
	whenTerms WhenTerms
	elseTerm  Expression
}

func NewSearchedCase(whenTerms WhenTerms, elseTerm Expression) Expression {
	return &SearchedCase{
		whenTerms: whenTerms,
		elseTerm:  elseTerm,
	}
}

func (this *SearchedCase) Evaluate(item value.Value, context Context) (value.Value, error) {
	for _, w := range this.whenTerms {
		wv, e := w.When.Evaluate(item, context)
		if e != nil {
			return nil, e
		}

		if wv.Truth() {
			tv, e := w.Then.Evaluate(item, context)
			if e != nil {
				return nil, e
			}

			return tv, nil
		}
	}

	if this.elseTerm == nil {
		return value.NULL_VALUE, nil
	}

	ev, e := this.elseTerm.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	return ev, nil
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

func (this *SearchedCase) VisitChildren(visitor Visitor) (Expression, error) {
	var e error
	for _, w := range this.whenTerms {
		w.When, e = visitor.Visit(w.When)
		if e != nil {
			return nil, e
		}

		w.Then, e = visitor.Visit(w.Then)
		if e != nil {
			return nil, e
		}
	}

	if this.elseTerm != nil {
		this.elseTerm, e = visitor.Visit(this.elseTerm)
		if e != nil {
			return nil, e
		}
	}

	return this, nil
}
