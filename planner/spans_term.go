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
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

type TermSpans struct {
	spans plan.Spans2
}

func NewTermSpans(spans ...*plan.Span2) *TermSpans {
	rv := &TermSpans{
		spans: spans,
	}

	return rv
}

func (this *TermSpans) CreateScan(
	index datastore.Index, term *algebra.KeyspaceTerm, indexApiVersion int,
	reverse, distinct, overlap, array bool, offset, limit expression.Expression,
	projection *plan.IndexProjection, indexOrder plan.IndexKeyOrders,
	indexGroupAggs *plan.IndexGroupAggregates, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value,
	filters base.Filters, cost, cardinality float64) plan.SecondaryScan {

	distScan := this.CanHaveDuplicates(index, indexApiVersion, overlap, array)

	if index3, ok := index.(datastore.Index3); ok && useIndex3API(index, indexApiVersion) {
		dynamicIn := this.spans.HasDynamicIn()
		if (filters != nil) && (cost > 0.0) && (cardinality > 0.0) {
			var err error
			keys := index.RangeKey().Copy()
			condition := index.Condition()
			if condition != nil {
				condition = condition.Copy()
			}
			if len(keys) > 0 || condition != nil {
				formalizer := expression.NewSelfFormalizer(term.Alias(), nil)

				for i, key := range keys {
					key = key.Copy()

					formalizer.SetIndexScope()
					key, err = formalizer.Map(key)
					formalizer.ClearIndexScope()
					if err != nil {
						break
					}

					keys[i] = key
				}

				if condition != nil && err == nil {
					condition, err = formalizer.Map(condition)
				}
			}
			if index.IsPrimary() {
				meta := expression.NewMeta(expression.NewIdentifier(term.Alias()))
				keys = append(keys, meta)
			}
			if err != nil {
				cost = OPT_COST_NOT_AVAIL
				cardinality = OPT_CARD_NOT_AVAIL
			} else {
				optMarkIndexFilters(keys, this.spans, condition, filters)
			}
		}
		if distScan && indexGroupAggs == nil {
			scan := plan.NewIndexScan3(index3, term, this.spans, reverse, false, dynamicIn, nil, nil,
				projection, indexOrder, indexGroupAggs, covers, filterCovers, cost, cardinality)

			if cost > 0.0 && cardinality > 0.0 {
				distCost, distCard := getDistinctScanCost(index, cardinality)
				if distCost > 0.0 && distCard > 0.0 {
					cost += distCost
					cardinality = distCard
				} else {
					cost = OPT_COST_NOT_AVAIL
					cardinality = OPT_CARD_NOT_AVAIL
				}
			}
			return plan.NewDistinctScan(limit, offset, scan, cost, cardinality)
		} else {
			return plan.NewIndexScan3(index3, term, this.spans, reverse, distinct, dynamicIn, offset, limit,
				projection, indexOrder, indexGroupAggs, covers, filterCovers, cost, cardinality)
		}

	} else if index2, ok := index.(datastore.Index2); ok && useIndex2API(index, indexApiVersion) {
		if !this.Exact() {
			limit = nil
			offset = nil
		}

		if distScan {
			scan := plan.NewIndexScan2(index2, term, this.spans, reverse, false, false, nil, nil,
				projection, covers, filterCovers)
			return plan.NewDistinctScan(limit, offset, scan, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL)
		} else {
			return plan.NewIndexScan2(index2, term, this.spans, reverse, distinct, false, offset, limit,
				projection, covers, filterCovers)
		}
	} else {
		var limitOffset expression.Expression

		spans, exact := ConvertSpans2ToSpan(this.spans, len(index.RangeKey()))
		if !exact {
			limit = nil
			offset = nil
		} else if exact || distScan {
			limitOffset = offsetPlusLimit(offset, limit)
		}

		if distScan || (len(spans) > 1 && !exact) {
			scan := plan.NewIndexScan(index, term, spans, distinct, limitOffset, covers, filterCovers)
			return plan.NewDistinctScan(limit, offset, scan, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL)
		} else {
			return plan.NewIndexScan(index, term, spans, distinct, limitOffset, covers, filterCovers)
		}
	}
}

func (this *TermSpans) Compose(prev SargSpans) SargSpans {
	return prev.Copy().ComposeTerm(this)
}

func (this *TermSpans) ComposeTerm(next *TermSpans) SargSpans {
	return composeTerms(this, next)
}

func (this *TermSpans) Constrain(other SargSpans) SargSpans {
	return other.Copy().ConstrainTerm(this)
}

func (this *TermSpans) ConstrainTerm(spans *TermSpans) SargSpans {
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

func (this *TermSpans) ExactSpan1(nkeys int) bool {
	for _, s := range this.spans {
		if !s.Exact {
			return false
		}
	}
	_, exact := ConvertSpans2ToSpan(this.spans, nkeys)

	return exact
}

func (this *TermSpans) SetExact(exact bool) {
	for _, s := range this.spans {
		s.Exact = exact
	}
}

func (this *TermSpans) CanUseIndexOrder(allowMultipleSpans bool) bool {
	return len(this.spans) == 1 || allowMultipleSpans
}

func (this *TermSpans) CanPushDownOffset(index datastore.Index, overlap, array bool) bool {
	return this.Exact() /* && !this.CanHaveDuplicates(index, overlap, array) */
}

func (this *TermSpans) CanHaveDuplicates(index datastore.Index, indexApiVersion int, overlap, array bool) bool {
	if useIndex3API(index, indexApiVersion) {
		return !array && indexHasArrayIndexKey(index)
	} else {
		return (len(this.spans) > 1 && (overlap || !this.Exact())) || (!array && indexHasArrayIndexKey(index))
	}
}

func (this *TermSpans) CanProduceUnknowns(pos int) bool {
	for _, span := range this.spans {
		if pos >= len(span.Ranges) {
			return true
		}

		range2 := span.Ranges[pos]
		low := range2.Low
		if low == nil || low.Type() < value.NULL || (low.Type() == value.NULL && (range2.Inclusion&datastore.LOW) != 0) {
			return true
		}
	}

	return false
}

func (this *TermSpans) SkipsLeadingNulls() bool {
	for _, span := range this.spans {
		if len(span.Ranges) == 0 {
			return false
		}

		range2 := span.Ranges[0]
		low := range2.Low
		if low == nil || low.Type() < value.NULL || (low.Type() == value.NULL && (range2.Inclusion&datastore.LOW) != 0) {
			return false
		}
	}

	return true
}

func (this *TermSpans) EquivalenceRangeAt(pos int) (eq bool, expr expression.Expression) {
	for i, span := range this.spans {
		if pos >= len(span.Ranges) || !span.Ranges[pos].EqualRange() {
			return false, nil
		}

		sexpr := span.Ranges[pos].Low
		if i > 0 && !expression.Equivalent(expr, sexpr) {
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

func (this *TermSpans) Spans() plan.Spans2 {
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

func composeTerms(rs, ns *TermSpans) SargSpans {
	// Cross product of prev and next spans
	sp := make(plan.Spans2, 0, len(rs.spans)*len(ns.spans))

	for _, prev := range rs.spans {
		if prev.Empty() {
			continue
		}

		pn := make(plan.Spans2, 0, len(ns.spans))
		for _, next := range ns.spans {
			if next.Empty() {
				continue
			}

			pre := prev.Copy()
			pre.Ranges = append(pre.Ranges, next.Ranges...)
			pre.Exact = pre.Exact || next.Exact
			pn = append(pn, pre)
		}
		if len(pn) != 0 {
			sp = append(sp, pn...)
		}
	}

	return NewTermSpans(sp...)
}

func constrainSpans(spans1, spans2 plan.Spans2) SargSpans {
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
	cspans := make(plan.Spans2, 0, len(spans1)*len(spans2))
	for _, span2 := range spans2 {
		copy1 := spans1.Copy()
		for _, span1 := range copy1 {
			constrainSpan(span1, span2)
		}
		cspans = append(cspans, copy1...)
	}

	return streamline(cspans)
}

func constrainSpan(span1, span2 *plan.Span2) {
	if span1.Exact && (!span2.Exact || constrainEmptySpan(span1, span2) || constrainEmptySpan(span2, span1)) {
		span1.Exact = false
	}

	// Adjust low bound
	if span2.Ranges[0].Low != nil {
		span1.Exact = span1.Exact && span2.Exact

		if span1.Ranges[0].Low == nil {
			// Get low bound from span2
			span1.Ranges[0].Low = span2.Ranges[0].Low
			span1.Ranges[0].Inclusion = (span1.Ranges[0].Inclusion & datastore.HIGH) |
				(span2.Ranges[0].Inclusion & datastore.LOW)
			span1.Ranges[0].Selec1 = span2.Ranges[0].Selec1
			span1.Ranges[0].SetCheckSpecialSpan()
			span1.Ranges[0].InheritFlags(span2.Ranges[0])
		} else {
			// Keep the greater or unknown low bound from
			// span1 and span2

			low1 := span1.Ranges[0].Low.Value()
			low2 := span2.Ranges[0].Low.Value()

			if span1.Exact {
				if low1 == nil && low2 == nil {
					span1.Exact = false
				} else if low1 == nil && (low2.Type() > value.NULL || (span2.Ranges[0].Inclusion&datastore.LOW) != 0) {
					// query parameter, non inclusive null
					span1.Exact = false
				} else if low2 == nil && (low1.Type() > value.NULL || (span1.Ranges[0].Inclusion&datastore.LOW) != 0) {
					// non inclusive null, query paramtere
					span1.Exact = false
				}
			}

			var res int
			if low1 != nil && low2 != nil {
				res = low1.Collate(low2)
			}

			if low1 != nil && (low2 == nil || res < 0) {
				span1.Ranges[0].Low = span2.Ranges[0].Low
				span1.Ranges[0].Inclusion = (span1.Ranges[0].Inclusion & datastore.HIGH) |
					(span2.Ranges[0].Inclusion & datastore.LOW)
				span1.Ranges[0].Selec1 = span2.Ranges[0].Selec1
				span1.Ranges[0].SetCheckSpecialSpan()
				span1.Ranges[0].InheritFlags(span2.Ranges[0])
			} else if low1 != nil && low2 != nil && res == 0 {
				span1.Ranges[0].Inclusion = (span1.Ranges[0].Inclusion & datastore.HIGH) |
					(span1.Ranges[0].Inclusion & span2.Ranges[0].Inclusion & datastore.LOW)
				span1.Ranges[0].SetCheckSpecialSpan()
				span1.Ranges[0].InheritFlags(span2.Ranges[0])
			}
		}
	}
	// Adjust high bound
	if span2.Ranges[0].High != nil {
		span1.Exact = span1.Exact && span2.Exact

		if span1.Ranges[0].High == nil {
			// Get high bound from span2

			span1.Ranges[0].High = span2.Ranges[0].High
			span1.Ranges[0].Inclusion = (span1.Ranges[0].Inclusion & datastore.LOW) |
				(span2.Ranges[0].Inclusion & datastore.HIGH)
			span1.Ranges[0].Selec2 = span2.Ranges[0].Selec2
			span1.Ranges[0].SetCheckSpecialSpan()
			span1.Ranges[0].InheritFlags(span2.Ranges[0])
		} else {
			// Keep the lesser or unknown high bound from
			// span1 and span2

			high1 := span1.Ranges[0].High.Value()
			high2 := span2.Ranges[0].High.Value()

			if span1.Exact && (high1 == nil || high2 == nil) {
				span1.Exact = false
			}

			var res int
			if high1 != nil && high2 != nil {
				res = high1.Collate(high2)
			}
			if high1 != nil && (high2 == nil || res > 0) {
				span1.Ranges[0].High = span2.Ranges[0].High
				span1.Ranges[0].Inclusion = (span1.Ranges[0].Inclusion & datastore.LOW) |
					(span2.Ranges[0].Inclusion & datastore.HIGH)
				span1.Ranges[0].Selec2 = span2.Ranges[0].Selec2
				span1.Ranges[0].SetCheckSpecialSpan()
				span1.Ranges[0].InheritFlags(span2.Ranges[0])
			} else if high1 != nil && high2 != nil && res == 0 {
				span1.Ranges[0].Inclusion = (span1.Ranges[0].Inclusion & datastore.LOW) |
					(span1.Ranges[0].Inclusion & span2.Ranges[0].Inclusion & datastore.HIGH)
				span1.Ranges[0].SetCheckSpecialSpan()
				span1.Ranges[0].InheritFlags(span2.Ranges[0])
			}
		}
	}
}

func constrainEmptySpan(span1, span2 *plan.Span2) bool {
	// handle empty span for f1 >= 3 and f1 < 3, f1 < 3 and f1 >= 3

	if span1.Ranges[0].High == nil || span2.Ranges[0].Low == nil {
		return false
	}

	// span1 HIGH, span2 LOW are set, so it will not empty span
	if (span1.Ranges[0].Inclusion&datastore.HIGH) != 0 && (span2.Ranges[0].Inclusion&datastore.LOW) != 0 {
		return false
	}

	// span1 HIGH, span2 LOW are not set, so it will not empty span
	if (span1.Ranges[0].Inclusion&datastore.HIGH) == 0 && (span2.Ranges[0].Inclusion&datastore.LOW) == 0 {
		return false
	}

	high1 := span1.Ranges[0].High.Value()
	low2 := span2.Ranges[0].Low.Value()
	if low2 != nil && high1 != nil && high1.Equals(low2).Truth() {
		return true
	}
	return false
}

func streamline(cspans plan.Spans2) SargSpans {
	switch len(cspans) {
	case 0:
		return _EMPTY_SPANS
	case 1:
		if cspans[0].Empty() {
			return _EMPTY_SPANS
		}
		checkSpecialSpan(cspans[0])
		return NewTermSpans(cspans...)
	}

	hash := _STRING_SPAN_POOL.Get()
	defer _STRING_SPAN_POOL.Put(hash)

	spans := make(plan.Spans2, 0, len(cspans))
	for _, cspan := range cspans {
		str := cspan.String()
		if _, found := hash[str]; !found && !cspan.Empty() {
			hash[str] = cspan
			spans = append(spans, cspan)
			checkSpecialSpan(cspan)
		}
	}

	if len(spans) == 0 {
		return _EMPTY_SPANS
	}

	for _, span := range spans {
		if span.EquivalentTo(_EXACT_FULL_SPAN) || span.EquivalentTo(_FULL_SPAN) ||
			span.EquivalentTo(_WHOLE_SPAN) {
			return NewTermSpans(span)
		}
	}

	return NewTermSpans(spans...)
}

func checkSpecialSpan(span *plan.Span2) {
	if span == nil {
		return
	}

	// only the first range is ever affected
	rg := span.Ranges[0]
	if rg.HasCheckSpecialSpan() {
		rg.ClearSpecialSpan()
		setSpecialSpan(rg)
		rg.UnsetCheckSpecialSpan()
	}
}

func setSpecialSpan(rg *plan.Range2) {
	if rg.EquivalentTo(_SELF_SPAN.Ranges[0]) {
		rg.Flags |= plan.RANGE_SELF_SPAN
	} else if rg.EquivalentTo(_FULL_SPAN.Ranges[0]) {
		rg.Flags |= plan.RANGE_FULL_SPAN
	} else if rg.EquivalentTo(_WHOLE_SPAN.Ranges[0]) {
		rg.Flags |= plan.RANGE_WHOLE_SPAN
	} else if rg.EquivalentTo(_VALUED_SPAN.Ranges[0]) {
		rg.Flags |= plan.RANGE_VALUED_SPAN
	} else if rg.EquivalentTo(_EMPTY_SPAN.Ranges[0]) {
		rg.Flags |= plan.RANGE_EMPTY_SPAN
	} else if rg.EquivalentTo(_NULL_SPAN.Ranges[0]) {
		rg.Flags |= plan.RANGE_NULL_SPAN
	} else if rg.EquivalentTo(_MISSING_SPAN.Ranges[0]) {
		rg.Flags |= plan.RANGE_MISSING_SPAN
	}
}

func ConvertSpans2ToSpan(spans2 plan.Spans2, total int) (plan.Spans, bool) {
	exact := true
	spans := make(plan.Spans, 0, len(spans2))
	for _, span2 := range spans2 {
		sp := &plan.Span{}
		sp.Range.Low = make([]expression.Expression, 0, len(span2.Ranges))
		sp.Range.High = make([]expression.Expression, 0, len(span2.Ranges))
		sp.Exact = span2.Exact

		addLow := true
		addHigh := true
		lowIncl := (datastore.LOW & span2.Ranges[0].Inclusion)
		highIncl := (datastore.HIGH & span2.Ranges[0].Inclusion)

		length := len(span2.Ranges)
		for i, range2 := range span2.Ranges {
			if range2.Low == nil {
				addLow = false
			}

			if addLow {
				sp.Range.Low = append(sp.Range.Low, range2.Low)
				lowIncl &= (datastore.LOW & range2.Inclusion)
			}

			if range2.High == nil {
				addHigh = false
			}

			if addHigh {
				sp.Range.High = append(sp.Range.High, range2.High)
				highIncl &= (datastore.HIGH & range2.Inclusion)
			}

			if sp.Exact && (i < length-1) && !range2.EqualRange() {
				sp.Exact = false
			}
		}

		i := len(sp.Range.High)
		if i > 0 && i < total && (span2.Ranges[i-1].Inclusion&datastore.HIGH) == datastore.HIGH {
			sp.Range.High[i-1] = expression.NewSuccessor(sp.Range.High[i-1])
			highIncl = datastore.NEITHER
		} else if i == 0 {
			sp.Range.High = nil
		}

		if len(sp.Range.Low) == 0 {
			sp.Range.Low = nil
		}

		sp.Range.Inclusion = (lowIncl | highIncl)
		exact = exact && sp.Exact
		spans = append(spans, sp)
	}

	if !exact {
		for _, sp := range spans {
			sp.Exact = exact
		}
	}

	return spans, exact
}

var _STRING_SPAN_POOL = plan.NewStringSpanPool(1024)
