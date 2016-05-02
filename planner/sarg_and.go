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
	"github.com/couchbase/query/util"
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
			if s[0] == _EMPTY_SPANS[0] {
				spans = _EMPTY_SPANS
				return
			}

			if len(spans) == 0 {
				spans = s.Copy()
			} else {
				spans = constrainSpans(spans, s)
				if spans[0] == _EMPTY_SPANS[0] {
					spans = _EMPTY_SPANS
					return
				}
			}
		}

		return
	}

	return rv
}

func constrainSpans(spans1, spans2 plan.Spans) plan.Spans {
	if len(spans2) > 1 && len(spans1) <= 1 {
		spans1, spans2 = spans2.Copy(), spans1
	}

	// Avoid copying if possible
	if len(spans2) <= 1 {
		for _, span2 := range spans2 {
			for _, span1 := range spans1 {
				constrainSpan(span1, span2)
			}
		}

		return deDupDiscardEmptySpans(spans1)
	}

	// Generate cross product of inputs
	cspans := make(plan.Spans, 0, len(spans1)*len(spans2))
	for _, span2 := range spans2 {
		copy1 := spans1.Copy()
		for _, span1 := range copy1 {
			constrainSpan(span1, span2)
		}
		cspans = append(cspans, copy1...)
	}

	return deDupDiscardEmptySpans(cspans)
}

func constrainSpan(span1, span2 *plan.Span) {

	if span1.Exact && (!span2.Exact || constrainEmptySpan(span1, span2) || constrainEmptySpan(span2, span1)) {
		span1.Exact = false
	}

	// Adjust low bound
	if len(span2.Range.Low) > 0 {
		span1.Exact = span1.Exact && span2.Exact

		if len(span1.Range.Low) == 0 {
			// Get low bound from span2

			span1.Range.Low = span2.Range.Low
			span1.Range.Inclusion = (span1.Range.Inclusion & datastore.HIGH) |
				(span2.Range.Inclusion & datastore.LOW)
		} else {
			// Keep the greater or unknown low bound from
			// span1 and span2

			low1 := span1.Range.Low[0].Value()
			low2 := span2.Range.Low[0].Value()

			if span1.Exact && (low1 == nil || low2 == nil) {
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
			} else if low1 != nil && low2 != nil && res == 0 {
				span1.Range.Inclusion = (span1.Range.Inclusion & datastore.HIGH) |
					(span1.Range.Inclusion & span2.Range.Inclusion & datastore.LOW)
			}
		}
	}

	// Adjust high bound
	if len(span2.Range.High) > 0 {
		span1.Exact = span1.Exact && span2.Exact

		if len(span1.Range.High) == 0 {
			// Get high bound from span2

			span1.Range.High = span2.Range.High
			span1.Range.Inclusion = (span1.Range.Inclusion & datastore.LOW) |
				(span2.Range.Inclusion & datastore.HIGH)
		} else {
			// Keep the lesser or unknown high bound from
			// span1 and span2

			high1 := span1.Range.High[0].Value()
			high2 := span2.Range.High[0].Value()

			if span1.Exact && (high1 == nil || high2 == nil) {
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
			} else if high1 != nil && high2 != nil && res == 0 {
				span1.Range.Inclusion = (span1.Range.Inclusion & datastore.LOW) |
					(span1.Range.Inclusion & span2.Range.Inclusion & datastore.HIGH)
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

/*
False negatives allowed.
*/
func isEmptySpan(span *plan.Span) bool {
	if span == _EMPTY_SPANS[0] {
		return true
	}
	low := span.Range.Low
	high := span.Range.High
	n := util.MinInt(len(low), len(high))

	for i := 0; i < n; i++ {
		lv := low[i].Value()
		hv := high[i].Value()
		if lv == nil || hv == nil {
			return false
		}

		c := lv.Collate(hv)
		if c == 0 {
			continue
		}
		return c > 0
	}

	return (len(low) == len(high) && (span.Range.Inclusion&datastore.BOTH) == 0)
}
