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

type sargAnd struct {
	sargBase
}

func newSargAnd(pred *expression.And) *sargAnd {
	rv := &sargAnd{}
	rv.sarger = func(expr2 expression.Expression) (spans plan.Spans, err error) {
		if SubsetOf(pred, expr2) {
			return _SELF_SPANS, nil
		}

		var s plan.Spans
		for _, op := range pred.Operands() {

			s, err = sargFor(op, expr2, rv.MissingHigh())
			if err != nil {
				return nil, err
			}

			if len(s) == 0 {
				continue
			}

			if len(spans) == 0 {
				spans = s.Copy()
			} else {
				spans = constrainSpans(spans, s)
			}
		}

		return
	}

	return rv
}

func constrainSpans(spans1, spans2 plan.Spans) plan.Spans {
	if len(spans2) != 1 {
		if len(spans1) == 1 {
			spans1, spans2 = spans2.Copy(), spans1
		} else {
			spans1[0].Exact = false
			return spans1
		}
	}

	span2 := spans2[0]
	for _, span1 := range spans1 {
		constrainSpan(span1, span2)
	}

	return spans1
}

func constrainSpan(span1, span2 *plan.Span) {

	if span1.Exact && (!span2.Exact || constrainEmptySpan(span1, span2) || constrainEmptySpan(span2, span1)) {
		span1.Exact = false
	}

	if len(span2.Range.Low) > 0 {
		if len(span1.Range.Low) == 0 {
			span1.Range.Low = span2.Range.Low
			span1.Range.Inclusion = (span1.Range.Inclusion & datastore.HIGH) |
				(span2.Range.Inclusion & datastore.LOW)
		} else {
			low1 := span1.Range.Low[0].Value()
			low2 := span2.Range.Low[0].Value()

			/* For query parameters exact span will be false
			   Span will be query parameter
			*/
			if low1 == nil || low2 == nil {
				span1.Exact = false
			}

			var res int
			if low1 != nil && low2 != nil {
				res = low1.Collate(low2)
			}

			if low1 != nil && (low2 == nil || res < 0) {
				span1.Range.Low = span2.Range.Low
				span1.Range.Inclusion = (span1.Range.Inclusion & datastore.HIGH) |
					(span2.Range.Inclusion & datastore.LOW)
			} else if low1 != nil && low2 != nil && res == 0 && (span1.Range.Inclusion&datastore.LOW) != 0 {
				span1.Range.Inclusion = (span1.Range.Inclusion & datastore.HIGH) |
					(span2.Range.Inclusion & datastore.LOW)

			}
		}
	}

	if len(span2.Range.High) > 0 {
		if len(span1.Range.High) == 0 {
			span1.Range.High = span2.Range.High
			span1.Range.Inclusion = (span1.Range.Inclusion & datastore.LOW) |
				(span2.Range.Inclusion & datastore.HIGH)
		} else {
			high1 := span1.Range.High[0].Value()
			high2 := span2.Range.High[0].Value()

			/* For query parameters exact span will be false
			   Span will be query parameter
			*/
			if high1 == nil || high2 == nil {
				span1.Exact = false
			}

			var res int
			if high1 != nil && high2 != nil {
				res = high1.Collate(high2)
			}

			if high1 != nil && (high2 == nil || res > 0) {
				span1.Range.High = span2.Range.High
				span1.Range.Inclusion = (span1.Range.Inclusion & datastore.LOW) |
					(span2.Range.Inclusion & datastore.HIGH)
			} else if high1 != nil && high2 != nil && res == 0 && (span1.Range.Inclusion&datastore.HIGH) != 0 {
				span1.Range.Inclusion = (span1.Range.Inclusion & datastore.LOW) |
					(span2.Range.Inclusion & datastore.HIGH)
			}
		}
	}
}

func constrainEmptySpan(span1, span2 *plan.Span) bool {
	// handle empty span for f1 >= 3 and f1 < 3, f1 < 3 and f1 >= 3

	if len(span1.Range.High) == 0 || len(span2.Range.Low) == 0 {
		return false
	}

	// span1 HIGH, span2 LOW are set, so it will not empty span
	if (span1.Range.Inclusion&datastore.HIGH) != 0 && (span2.Range.Inclusion&datastore.LOW) != 0 {
		return false
	}

	// span1 HIGH, span2 LOW are not set, so it will not empty span
	if (span1.Range.Inclusion&datastore.HIGH) == 0 && (span2.Range.Inclusion&datastore.LOW) == 0 {
		return false
	}

	high1 := span1.Range.High[0].Value()
	low2 := span2.Range.Low[0].Value()
	if low2 != nil && high1 != nil && high1.Equals(low2).Truth() {
		return true
	}
	return false
}
