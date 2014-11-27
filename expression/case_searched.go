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

func (this *SearchedCase) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSearchedCase(this)
}

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

func (this *SearchedCase) Indexable() bool {
	return this.indexable(this)
}

func (this *SearchedCase) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
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
