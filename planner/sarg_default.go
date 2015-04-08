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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
)

var _SELF_SPANS Spans
var _FULL_SPANS Spans
var _VALUED_SPANS Spans

func init() {
	sspan := &Span{}
	sspan.Range.Low = expression.Expressions{expression.TRUE_EXPR}
	sspan.Range.Inclusion = datastore.LOW
	_SELF_SPANS = Spans{sspan}

	fspan := &Span{}
	fspan.Range.Low = expression.Expressions{expression.NULL_EXPR}
	fspan.Range.Inclusion = datastore.LOW
	_FULL_SPANS = Spans{fspan}

	vspan := &Span{}
	vspan.Range.Low = expression.Expressions{expression.NULL_EXPR}
	vspan.Range.Inclusion = datastore.NEITHER
	_VALUED_SPANS = Spans{vspan}
}

type sargDefault struct {
	sargBase
}

func newSargDefault(cond expression.Expression) *sargDefault {
	var spans Spans
	if cond.PropagatesNull() {
		spans = _VALUED_SPANS
	} else if cond.PropagatesMissing() {
		spans = _FULL_SPANS
	}

	rv := &sargDefault{}
	rv.sarg = func(expr2 expression.Expression) (Spans, error) {
		if SubsetOf(cond, expr2) {
			return _SELF_SPANS, nil
		}

		if spans != nil && cond.DependsOn(expr2) {
			return spans, nil
		}

		return nil, nil
	}

	return rv
}
