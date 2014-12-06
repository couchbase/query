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

var _NULL_SPANS datastore.Spans

func init() {
	span := &datastore.Span{}
	span.Range.Low = value.Values{value.NULL_VALUE}
	span.Range.High = span.Range.Low
	span.Range.Inclusion = datastore.BOTH
	_NULL_SPANS = datastore.Spans{span}
}

type sargNull struct {
	sargBase
}

func newSargNull(expr *expression.IsNull) *sargNull {
	rv := &sargNull{}
	rv.sarg = func(expr2 expression.Expression) (datastore.Spans, error) {
		if expr.EquivalentTo(expr2) {
			return _SELF_SPANS, nil
		}

		if !expr.Operand().EquivalentTo(expr2) {
			return nil, nil
		}

		return _NULL_SPANS, nil
	}

	return rv
}

type sargNotNull struct {
	sargBase
}

func newSargNotNull(expr *expression.IsNotNull) *sargNotNull {
	rv := &sargNotNull{}
	rv.sarg = func(expr2 expression.Expression) (datastore.Spans, error) {
		if expr.EquivalentTo(expr2) {
			return _SELF_SPANS, nil
		}

		if !expr.Operand().EquivalentTo(expr2) {
			return nil, nil
		}

		return _VALUED_SPANS, nil
	}

	return rv
}
