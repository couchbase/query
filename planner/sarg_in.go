//  Copyright (c) 2016 Couchbase, Inc.
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
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/value"
)

func (this *sarg) VisitIn(pred *expression.In) (interface{}, error) {
	if SubsetOf(pred, this.key) {
		return _SELF_SPANS, nil
	}

	if !SubsetOf(pred.First(), this.key) {
		if pred.DependsOn(this.key) {
			return _VALUED_SPANS, nil
		} else {
			return nil, nil
		}
	}

	var array expression.Expressions

	aval := pred.Second().Value()
	if aval != nil {
		vals, ok := aval.Actual().([]interface{})
		if !ok || len(vals) == 0 {
			return _EMPTY_SPANS, nil
		}

		// De-dup before generating spans
		set := value.NewSet(len(vals), true)
		set.AddAll(vals)
		vals = set.Actuals()

		// Sort for EXPLAIN stability
		sort.Sort(value.NewSorter(value.NewValue(vals)))

		array = make(expression.Expressions, len(vals))
		for i, val := range vals {
			array[i] = expression.NewConstant(val)
		}
	}

	if array == nil {
		second := pred.Second()
		if acons, ok := second.(*expression.ArrayConstruct); ok {
			array = acons.Operands()
		} else if second.Static() == nil {
			return _VALUED_SPANS, nil
		} else {
			static := second.Static()

			span := &plan.Span{}
			span.Range.Low = expression.Expressions{expression.NewArrayMin(static)}
			if this.missingHigh {
				span.Range.High = expression.Expressions{
					expression.NewSuccessor(expression.NewArrayMax(static))}
				span.Range.Inclusion = datastore.LOW
			} else {
				span.Range.High = expression.Expressions{expression.NewArrayMax(static)}
				span.Range.Inclusion = datastore.BOTH
			}

			span.Exact = false
			return NewTermSpans(span), nil
		}
	}

	if len(array) == 0 {
		return _EMPTY_SPANS, nil
	}

	spans := make(plan.Spans, 0, len(array))
	for _, elem := range array {
		static := elem.Static()
		if static == nil {
			return _VALUED_SPANS, nil
		}

		val := static.Value()
		if val != nil && val.Type() <= value.NULL {
			continue
		}

		span := &plan.Span{}
		span.Range.Low = expression.Expressions{static}
		if this.missingHigh {
			span.Range.High = expression.Expressions{expression.NewSuccessor(span.Range.Low[0])}
			span.Range.Inclusion = datastore.LOW
		} else {
			span.Range.High = span.Range.Low
			span.Range.Inclusion = datastore.BOTH
		}

		span.Exact = (val != nil)
		spans = append(spans, span)
	}

	if len(spans) == 0 {
		return _EMPTY_SPANS, nil
	}

	return NewTermSpans(spans...), nil
}
