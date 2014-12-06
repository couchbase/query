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
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type sargDefault struct {
	sargBase
}

var _SELF_SPANS datastore.Spans
var _FULL_SPANS datastore.Spans
var _VALUED_SPANS datastore.Spans

func init() {
	sspan := &datastore.Span{}
	sspan.Range.Low = value.Values{value.TRUE_VALUE}
	sspan.Range.Inclusion = datastore.LOW
	_SELF_SPANS = datastore.Spans{sspan}

	fspan := &datastore.Span{}
	fspan.Range.Low = value.Values{value.NULL_VALUE}
	fspan.Range.Inclusion = datastore.LOW
	_FULL_SPANS = datastore.Spans{fspan}

	vspan := &datastore.Span{}
	vspan.Range.Low = value.Values{value.FALSE_VALUE}
	vspan.Range.Inclusion = datastore.LOW
	_VALUED_SPANS = datastore.Spans{vspan}
}

func newSargDefault(expr expression.Expression) *sargDefault {
	var spans datastore.Spans
	if expr.PropagatesNull() {
		spans = _VALUED_SPANS
	} else if expr.PropagatesMissing() {
		spans = _FULL_SPANS
	}

	rv := &sargDefault{}
	rv.sarg = func(expr2 expression.Expression) (datastore.Spans, error) {
		if expr.EquivalentTo(expr2) {
			return _SELF_SPANS, nil
		}

		if spans != nil && expr.DependsOn(expr2) {
			return _VALUED_SPANS, nil
		}

		if spans != nil && expr.DependsOn(expr2) {
			return _FULL_SPANS, nil
		}

		return nil, nil
	}

	return rv
}
