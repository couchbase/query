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
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type TermSpans struct {
	spans plan.Spans
}

func NewTermSpans(spans ...*plan.Span) *TermSpans {
	rv := &TermSpans{
		spans: spans,
	}

	return rv
}

func (this *TermSpans) CreateScan(
	index datastore.Index, term *algebra.KeyspaceTerm, distinct, overlap,
	array bool, limit expression.Expression, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value) plan.SecondaryScan {

	exact := this.Exact()
	if !exact {
		limit = nil
	}

	if (len(this.spans) > 1 && (overlap || !exact)) ||
		(!array && indexHasArrayIndexKey(index)) {
		scan := plan.NewIndexScan(index, term, this.spans, distinct, nil, covers, filterCovers)
		return plan.NewDistinctScan(limit, scan)
	} else {
		return plan.NewIndexScan(index, term, this.spans, distinct, limit, covers, filterCovers)
	}
}

func (this *TermSpans) Compose(prev SargSpans) SargSpans {
	return prev.Copy().ComposeSpans(this)
}

func (this *TermSpans) ComposeSpans(next *TermSpans) SargSpans {
	return composeSpans(this, next)
}

func (this *TermSpans) Constrain(other SargSpans) SargSpans {
	return other.Copy().ConstrainSpans(this)
}

func (this *TermSpans) ConstrainSpans(spans *TermSpans) SargSpans {
	return constrainSpans(this.spans, spans.spans)
}

func (this *TermSpans) Streamline() SargSpans {
	return streamline(this.spans)
}

func (this *TermSpans) Exact() bool {
	for _, s := range this.spans {
		if !s.Exact {
			return false
		}
	}

	return true
}

func (this *TermSpans) SetExact(exact bool) {
	for _, s := range this.spans {
		s.Exact = exact
	}
}

func (this *TermSpans) SetExactForComposite(sargLength int) bool {
	exact := true

	// Except last key all leading keys needs to be EQ
	for _, span := range this.spans {
		for i := 0; i < sargLength-1; i++ {
			span.Exact = span.Exact && equalRangeKey(i, span.Range.Low, span.Range.High)
			exact = exact && span.Exact
		}
	}

	return exact
}

func (this *TermSpans) MissingHigh() bool {
	for _, span := range this.spans {
		if len(span.Range.High) == 0 {
			return true
		}
	}

	return false
}

func (this *TermSpans) CanUseIndexOrder() bool {
	return len(this.spans) == 1
}

func (this *TermSpans) SkipsLeadingNulls() bool {
	for _, span := range this.spans {
		if len(span.Range.Low) == 0 {
			return false
		}

		low := span.Range.Low[0]
		if low == nil ||
			low.Type() < value.NULL ||
			(low.Type() == value.NULL && (span.Range.Inclusion&datastore.LOW) != 0) {
			return false
		}
	}

	return true
}

func (this *TermSpans) EquivalenceRangeAt(i int) (eq bool, expr expression.Expression) {
	for _, span := range this.spans {
		if !equalRangeKey(i, span.Range.Low, span.Range.High) {
			return false, nil
		}

		sexpr := span.Range.Low[i]

		if (sexpr == nil) || (expr != nil && !sexpr.EquivalentTo(expr)) {
			return false, nil
		}

		expr = sexpr
	}

	return true, expr
}

func (this *TermSpans) Size() int {
	return len(this.spans)
}

func (this *TermSpans) Copy() SargSpans {
	rv := &TermSpans{
		spans: this.spans.Copy(),
	}

	return rv
}

func (this *TermSpans) Spans() plan.Spans {
	return this.spans
}

func (this *TermSpans) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *TermSpans) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{
		"#":     "TermSpans",
		"spans": this.spans,
	}

	return json.Marshal(r)
}

func composeSpans(rs, ns *TermSpans) SargSpans {
	// Cross product of prev and next spans
	sp := make(plan.Spans, 0, len(rs.spans)*len(ns.spans))

	for _, prev := range rs.spans {
		// Full span subsumes others
		if ns == _FULL_SPANS || ns == _EXACT_FULL_SPANS {
			prev.Exact = false
			sp = append(sp, prev)
			continue
		}

		pn := make(plan.Spans, 0, len(ns.spans))
		for _, next := range ns.spans {
			add := false
			pre := prev.Copy()

			if len(pre.Range.Low) > 0 && len(next.Range.Low) > 0 {
				pre.Range.Low = append(pre.Range.Low, next.Range.Low...)

				pre.Range.Inclusion = (datastore.LOW & pre.Range.Inclusion & next.Range.Inclusion) |
					(datastore.HIGH & pre.Range.Inclusion)
				add = true
			} else if len(next.Range.Low) > 0 {
				pre.Exact = false
			}

			if len(pre.Range.High) > 0 && len(next.Range.High) > 0 {
				pre.Range.High = append(pre.Range.High, next.Range.High...)
				pre.Range.Inclusion = (datastore.HIGH & pre.Range.Inclusion & next.Range.Inclusion) |
					(datastore.LOW & pre.Range.Inclusion)
				add = true
			} else if len(next.Range.High) > 0 {
				// f1 >=3 and f2 = 2 become span of {[3, 2] [] 1}, high of f2 is missing
				pre.Exact = false
			}

			// TODO: In Spock API2, all will be added
			if add {
				pn = append(pn, pre)
			} else {
				break
			}
		}

		// TODO: In Spock API2, all will be added
		if len(pn) == len(ns.spans) {
			sp = append(sp, pn...)
		} else {
			prev.Exact = false
			sp = append(sp, prev)
		}
	}

	return NewTermSpans(sp...)
}

func constrainSpans(spans1, spans2 plan.Spans) SargSpans {
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

		return streamline(spans1)
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

	return streamline(cspans)
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

			if span1.Exact {
				if low1 == nil && low2 == nil {
					span1.Exact = false
				} else if low1 == nil && (low2.Type() > value.NULL || (span2.Range.Inclusion&datastore.LOW) != 0) {
					// query parameter, non inclusive null
					span1.Exact = false
				} else if low2 == nil && (low1.Type() > value.NULL || (span1.Range.Inclusion&datastore.LOW) != 0) {
					// non inclusive null, query paramtere
					span1.Exact = false
				}
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
	if span == _EMPTY_SPAN {
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

	return (len(low) == len(high) && span.Range.Inclusion == datastore.NEITHER)
}

func streamline(cspans plan.Spans) SargSpans {
	switch len(cspans) {
	case 0:
		return _EMPTY_SPANS
	case 1:
		if isEmptySpan(cspans[0]) {
			return _EMPTY_SPANS
		}
		return NewTermSpans(cspans...)
	}

	hash := _STRING_SPAN_POOL.Get()
	defer _STRING_SPAN_POOL.Put(hash)

	spans := make(plan.Spans, 0, len(cspans))
	for _, cspan := range cspans {
		str := cspan.String()
		if _, found := hash[str]; !found && !isEmptySpan(cspan) {
			hash[str] = cspan
			spans = append(spans, cspan)
		}
	}

	if len(spans) == 0 {
		return _EMPTY_SPANS
	}

	for _, span := range spans {
		if span == _EXACT_FULL_SPAN ||
			(span.Exact && len(span.Range.Low) == 0 && len(span.Range.High) == 0) {
			return _EXACT_FULL_SPANS
		}

		if span == _FULL_SPAN ||
			(span.Exact && len(span.Range.Low) == 0 && len(span.Range.High) == 0) {
			return _FULL_SPANS
		}
	}

	return NewTermSpans(spans...)
}

func equalRangeKey(keyIndex int, low, high expression.Expressions) bool {
	if keyIndex >= len(low) || keyIndex >= len(high) {
		return false
	}

	if low[keyIndex] == high[keyIndex] {
		return true
	}

	if low[keyIndex] == nil || high[keyIndex] == nil {
		return false
	}

	var highExp expression.Expression
	switch hs := high[keyIndex].(type) {
	case *expression.Successor:
		highExp = hs.Operand()
	default:
		highExp = high[keyIndex]
	}

	return highExp.EquivalentTo(low[keyIndex])
}

var _STRING_SPAN_POOL = plan.NewStringSpanPool(1024)
