//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
)

// Shared base class
type multiSpansBase struct {
	spans []SargSpans
}

func (this *multiSpansBase) compose(prev SargSpans) {
	for i, s := range this.spans {
		this.spans[i] = s.Compose(prev)
	}
}

func (this *multiSpansBase) composeSpans(next *TermSpans) {
	for i, s := range this.spans {
		this.spans[i] = s.ComposeSpans(next)
	}
}

func (this *multiSpansBase) constrain(other SargSpans) {
	for i, s := range this.spans {
		this.spans[i] = s.Constrain(other)
	}
}

func (this *multiSpansBase) constrainSpans(spans *TermSpans) {
	for i, s := range this.spans {
		this.spans[i] = s.ConstrainSpans(spans)
	}
}

func (this *multiSpansBase) Exact() bool {
	for _, s := range this.spans {
		if !s.Exact() {
			return false
		}
	}

	return true
}

func (this *multiSpansBase) SetExact(exact bool) {
	for _, s := range this.spans {
		s.SetExact(exact)
	}
}

func (this *multiSpansBase) SetExactForComposite(sargLength int) bool {
	exact := true
	for _, s := range this.spans {
		exact = s.SetExactForComposite(sargLength) && exact
	}

	return exact
}

func (this *multiSpansBase) MissingHigh() bool {
	for _, s := range this.spans {
		if s.MissingHigh() {
			return true
		}
	}

	return false
}

func (this *multiSpansBase) EquivalenceRangeAt(i int) (eq bool, expr expression.Expression) {
	for _, s := range this.spans {
		seq, sexpr := s.EquivalenceRangeAt(i)

		if !seq || (expr != nil && !sexpr.EquivalentTo(expr)) {
			return false, nil
		}

		expr = sexpr
	}

	return expr != nil, expr
}

func dedupSpans(spans []SargSpans) []SargSpans {
	if len(spans) <= 1 {
		return spans
	}

	rv := make([]SargSpans, 0, len(spans))
	hash := _STRING_SPANS_POOL.Get()
	defer _STRING_SPANS_POOL.Put(hash)

	for _, span := range spans {
		s := span.String()
		if _, ok := hash[s]; !ok {
			hash[s] = true
			rv = append(rv, span)
		}
	}

	return rv
}

var _SPANS_POOL = NewSargSpansPool(16)
var _STRING_SPANS_POOL = util.NewStringBoolPool(16)
