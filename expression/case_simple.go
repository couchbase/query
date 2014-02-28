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

type WhenTerms []*WhenTerm

type WhenTerm struct {
	When Expression
	Then Expression
}

type SimpleCase struct {
	ExpressionBase
	searchTerm Expression
	whenTerms  WhenTerms
	elseTerm   Expression
}

func NewSimpleCase(searchTerm Expression, whenTerms WhenTerms, elseTerm Expression) Expression {
	return &SimpleCase{
		searchTerm: searchTerm,
		whenTerms:  whenTerms,
		elseTerm:   elseTerm,
	}
}

func (this *SimpleCase) Evaluate(item value.Value, context Context) (value.Value, error) {
	s, e := this.searchTerm.Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if s.Type() <= value.NULL {
		return s, nil
	}

	for _, w := range this.whenTerms {
		wv, e := w.When.Evaluate(item, context)
		if e != nil {
			return nil, e
		}

		if s.Equals(wv) {
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

func (this *SimpleCase) Dependencies() Expressions {
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

func (this *SimpleCase) Fold() Expression {
	this.searchTerm = this.searchTerm.Fold()

	for _, w := range this.whenTerms {
		w.When = w.When.Fold()
		w.Then = w.Then.Fold()
	}

	if this.elseTerm != nil {
		this.elseTerm = this.elseTerm.Fold()
	}

	return this
}
