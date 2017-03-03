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
	"github.com/couchbase/query/plan"
)

var _NULL_EXPRS = expression.Expressions{expression.NULL_EXPR}

var _NULL_SPANS SargSpans

func init() {
	span := &plan.Span{}
	span.Range.Low = expression.Expressions{expression.NULL_EXPR}
	span.Range.High = span.Range.Low
	span.Range.Inclusion = datastore.BOTH
	span.Exact = true
	_NULL_SPANS = NewTermSpans(span)
}

func (this *sarg) VisitIsNull(pred *expression.IsNull) (interface{}, error) {
	if SubsetOf(pred, this.key) {
		return _SELF_SPANS, nil
	}

	if pred.Operand().EquivalentTo(this.key) {
		return _NULL_SPANS, nil
	}

	var spans SargSpans
	if pred.Operand().PropagatesMissing() {
		spans = _FULL_SPANS
	}

	if spans != nil && pred.Operand().DependsOn(this.key) {
		return spans, nil
	}

	return nil, nil
}

func (this *sarg) VisitIsNotNull(pred *expression.IsNotNull) (interface{}, error) {
	if SubsetOf(pred, this.key) {
		return _SELF_SPANS, nil
	}

	if pred.Operand().EquivalentTo(this.key) {
		return _EXACT_VALUED_SPANS, nil
	}

	var spans SargSpans
	if pred.Operand().PropagatesNull() {
		spans = _VALUED_SPANS
	} else if pred.Operand().PropagatesMissing() {
		spans = _FULL_SPANS
	}

	if spans != nil && pred.Operand().DependsOn(this.key) {
		return spans, nil
	}

	return nil, nil
}
