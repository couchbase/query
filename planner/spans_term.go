//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	spans   plan.Spans2
	arrayId int
	annPos  int
	ann     *expression.Ann
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
	filterCovers map[*expression.Cover]value.Value, filter expression.Expression,
	cost, cardinality float64, size int64, frCost float64, includeSpans plan.Spans2,
	baseKeyspace *base.BaseKeyspace, hasDeltaKeyspace, skipNewKeys, nested_loop, setop bool,
	indexKeyNames []string, indexPartitionSets plan.IndexPartitionSets) plan.SecondaryScan {

	distScan := this.CanHaveDuplicates(index, indexApiVersion, overlap, array)

	if index3, ok := index.(datastore.Index3); ok && useIndex3API(index, indexApiVersion) {
		var indexVector *plan.IndexVector
		if index6, ok := index.(datastore.Index6); ok && useIndex6API(index, indexApiVersion) && !setop && this.ann != nil {
			var reRank expression.Expression
			if index6.AllowRerank() {
				reRank = this.ann.ReRank()
			}
			squareRoot := this.ann.NeedSquareRoot()
			indexVector = plan.NewIndexVector(this.ann.QueryVector(), this.annPos,
				this.ann.Nprobes(), reRank, squareRoot)
		} else {
			indexKeyNames = nil
			indexPartitionSets = nil
		}
		dynamicIn := this.spans.HasDynamicIn()
		if distScan && indexGroupAggs == nil {
			scan := plan.NewIndexScan3(index3, term, this.spans, includeSpans, reverse, false, dynamicIn,
				nil, nil, projection, indexOrder, indexGroupAggs, covers, filterCovers, filter,
				cost, cardinality, size, frCost, hasDeltaKeyspace, skipNewKeys, nested_loop,
				indexVector, indexKeyNames, indexPartitionSets)

			if cost > 0.0 && cardinality > 0.0 {
				distCost, distCard, distFrCost := getDistinctScanCost(index,
					cardinality, this.spans, baseKeyspace)
				if distCost > 0.0 && distCard > 0.0 && distFrCost > 0.0 {
					cost += distCost
					cardinality = distCard
					frCost += distFrCost
				} else {
					cost = OPT_COST_NOT_AVAIL
					cardinality = OPT_CARD_NOT_AVAIL
					size = OPT_SIZE_NOT_AVAIL
					frCost = OPT_COST_NOT_AVAIL
				}
			}
			return plan.NewDistinctScan(limit, offset, scan, cost, cardinality, size, frCost)
		} else {
			return plan.NewIndexScan3(index3, term, this.spans, includeSpans, reverse, distinct, dynamicIn,
				offset, limit, projection, indexOrder, indexGroupAggs, covers, filterCovers, filter,
				cost, cardinality, size, frCost, hasDeltaKeyspace, skipNewKeys, nested_loop,
				indexVector, indexKeyNames, indexPartitionSets)
		}

	} else if index2, ok := index.(datastore.Index2); ok && useIndex2API(index, indexApiVersion) {
		if !this.Exact() {
			limit = nil
			offset = nil
		}

		if distScan {
			scan := plan.NewIndexScan2(index2, term, this.spans, reverse, false, false, nil, nil,
				projection, covers, filterCovers, hasDeltaKeyspace, nested_loop)
			return plan.NewDistinctScan(limit, offset, scan, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_COST_NOT_AVAIL)
		} else {
			return plan.NewIndexScan2(index2, term, this.spans, reverse, distinct, false, offset, limit,
				projection, covers, filterCovers, hasDeltaKeyspace, nested_loop)
		}
	} else {
		var limitOffset expression.Expression

		spans, exact := ConvertSpans2ToSpan(this.Spans(), getIndexSize(index))
		if !exact {
			limit = nil
			offset = nil
		} else if exact || distScan {
			limitOffset = offsetPlusLimit(offset, limit)
		}

		if distScan || (len(spans) > 1 && !exact) {
			scan := plan.NewIndexScan(index, term, spans, distinct, limitOffset, covers, filterCovers, hasDeltaKeyspace,
				nested_loop)
			return plan.NewDistinctScan(limit, offset, scan, OPT_COST_NOT_AVAIL, OPT_CARD_NOT_AVAIL, OPT_SIZE_NOT_AVAIL,
				OPT_CARD_NOT_AVAIL)
		} else {
			return plan.NewIndexScan(index, term, spans, distinct, limitOffset, covers, filterCovers, hasDeltaKeyspace, nested_loop)
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

func (this *TermSpans) ConstrainTerm(other *TermSpans) SargSpans {
	rv := constrainSpans(this.spans, other.spans)
	return rv.inheritTermInfo(this, other)
}

func (this *TermSpans) Streamline() SargSpans {
	rv := streamline(this.spans)
	if this.arrayId != 0 || this.ann != nil {
		if rv.spans.HasStatic() {
			rv = rv.Copy().(*TermSpans)
		}
		rv.arrayId = this.arrayId
		rv.ann = this.ann
		rv.annPos = this.annPos
	}
	return rv
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

func (this *TermSpans) HasStatic() bool {
	return this.spans.HasStatic()
}

func (this *TermSpans) HasVector() bool {
	return this.ann != nil
}

func (this *TermSpans) SetExact(exact bool) {
	if len(this.spans) == 1 && this.HasStatic() {
		return
	}

	for pos, s := range this.spans {
		if s.Static {
			s = s.Copy()
			this.spans[pos] = s
		}
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

func (this *TermSpans) ValidRangeAt(pos int) bool {
	for _, span := range this.spans {
		if pos >= len(span.Ranges) {
			return false
		}
	}
	return true
}

func (this *TermSpans) Size() int {
	return len(this.spans)
}

func (this *TermSpans) Copy() SargSpans {
	rv := &TermSpans{
		spans:   this.spans.Copy(),
		arrayId: this.arrayId,
	}

	if this.ann != nil {
		rv.ann = this.ann.Copy().(*expression.Ann)
		rv.annPos = this.annPos
	}

	return rv
}

func (this *TermSpans) SetArrayId(id int) {
	if this.arrayId == 0 {
		this.arrayId = id
	}
}

func (this *TermSpans) ArrayId() int {
	return this.arrayId
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

func (this *TermSpans) inheritTermInfo(ts1, ts2 *TermSpans) *TermSpans {
	newArrayId := 0
	if ts1.arrayId != 0 && ts2.arrayId != 0 {
		if ts1.arrayId == ts2.arrayId {
			newArrayId = ts1.arrayId
		} else {
			// it's an error condition (unexpected), ignore arrayId
			newArrayId = -1
		}
	} else if ts1.arrayId != 0 {
		newArrayId = ts1.arrayId
	} else if ts2.arrayId != 0 {
		newArrayId = ts2.arrayId
	}

	var ann *expression.Ann
	var annPos int
	if ts1.ann != nil && ts2.ann != nil {
		if ts1.ann.EquivalentTo(ts2.ann) && ts1.annPos == ts2.annPos {
			ann = ts1.ann
			annPos = ts1.annPos
		} // else it's an error condition (unexpected), ignore ann
	} else if ts1.ann != nil {
		ann = ts1.ann
		annPos = ts1.annPos
	} else if ts2.ann != nil {
		ann = ts2.ann
		annPos = ts2.annPos
	}

	if (newArrayId != 0 && newArrayId != this.arrayId) || ann != nil {
		rv := this
		if this.spans.HasStatic() {
			rv = this.Copy().(*TermSpans)
		}
		if newArrayId != 0 && newArrayId != this.arrayId {
			rv.arrayId = newArrayId
		}
		if ann != nil {
			rv.ann = ann
			rv.annPos = annPos
		}
		return rv
	}
	return this
}

func composeTerms(rs, ns *TermSpans) *TermSpans {
	// Cross product of prev and next spans
	sp := make(plan.Spans2, 0, len(rs.spans)*len(ns.spans))

	// if we have multiple array predicate, do not try to combine spans from different array preds
	if rs.arrayId > 0 && ns.arrayId > 0 && rs.arrayId != ns.arrayId {
		ns = ns.Copy().(*TermSpans)
		for _, next := range ns.spans {
			if next.Empty() {
				continue
			}
			// Note this assumes that the array index keys are always next to each other,
			// i.e. FLATTEN_KEYS, thus the array index key from ns is always at Ranges[0].
			// Make Ranges[0] look like a _WHOLE_SPAN
			rg := next.Ranges[0]
			next.Ranges[0] = _WHOLE_SPAN.Ranges[0].Copy()
			if rg.Selec1 >= 0.0 || rg.Selec2 >= 0.0 {
				next.Ranges[0].Selec1 = 1.0
				next.Ranges[0].Selec2 = -1.0
			}
		}
		// reset arrayId since we replaced the span for array index key with _WHOLE_SPAN
		ns.arrayId = 0
	}

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
			pre.Exact = pre.Exact && next.Exact
			pn = append(pn, pre)
		}
		if len(pn) != 0 {
			sp = append(sp, pn...)
		}
	}

	rv := NewTermSpans(sp...)
	return rv.inheritTermInfo(rs, ns)
}

func constrainSpans(spans1, spans2 plan.Spans2) *TermSpans {
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
	// when this is called during sarging of index spans, each span has a single Range;
	// however it is possible to have multiple ranges during intersectSpan generation,
	// in which case constrain each individual range separately
	end := len(span1.Ranges)
	if end > len(span2.Ranges) {
		end = len(span2.Ranges)
	}
	for i := 0; i < end; i++ {
		range1 := span1.Ranges[i]
		range2 := span2.Ranges[i]

		if span1.Exact && (!span2.Exact || constrainEmptySpan(range1, range2) || constrainEmptySpan(range2, range1)) {
			span1.Exact = false
		}

		span1.Exact = constrainRange(range1, range2, span1.Exact, span2.Exact)
	}
}

func constrainRange(range1, range2 *plan.Range2, exact1, exact2 bool) bool {
	var changed bool

	// Adjust low bound
	if range2.Low != nil {
		exact1 = exact1 && exact2

		if range1.Low == nil {
			// Get low bound from range2
			range1.Low = range2.Low
			range1.Inclusion = (range1.Inclusion & datastore.HIGH) |
				(range2.Inclusion & datastore.LOW)
			range1.Selec1 = range2.Selec1
			changed = true
		} else {
			// Keep the greater or unknown low bound from
			// range1 and range2

			low1 := range1.Low.Value()
			low2 := range2.Low.Value()

			if exact1 {
				if low1 == nil && low2 == nil {
					exact1 = false
				} else if low1 == nil && (low2.Type() > value.NULL || (range2.Inclusion&datastore.LOW) != 0) {
					// query parameter, non inclusive null
					exact1 = false
				} else if low2 == nil && (low1.Type() > value.NULL || (range1.Inclusion&datastore.LOW) != 0) {
					// non inclusive null, query paramtere
					exact1 = false
				}
			}

			var use2, useBoth, useGreatest bool
			if low1 != nil && low2 != nil {
				res := low1.Collate(low2)

				if res < 0 {
					use2 = true
				} else if res == 0 {
					useBoth = true
				}
			} else if low1 != nil {
				if low1.Type() > value.NULL {
					useGreatest = true
				} else {
					use2 = true
				}
			} else if low2 != nil {
				if low2.Type() > value.NULL {
					useGreatest = true
				}
			}
			if useBoth {
				range1.Inclusion = (range1.Inclusion & datastore.HIGH) |
					(range1.Inclusion & range2.Inclusion & datastore.LOW)
				changed = true
			} else if useGreatest {
				exact1 = false
				range1.Low = getLowHigh(range1.Low, range2.Low, true)
				range1.Inclusion = (range1.Inclusion & datastore.HIGH) |
					((range1.Inclusion | range2.Inclusion) & datastore.LOW)
				changed = true
			} else if use2 {
				range1.Low = range2.Low
				range1.Inclusion = (range1.Inclusion & datastore.HIGH) |
					(range2.Inclusion & datastore.LOW)
				range1.Selec1 = range2.Selec1
				changed = true
			}
		}
	}
	// Adjust high bound
	if range2.High != nil {
		exact1 = exact1 && exact2

		if range1.High == nil {
			// Get high bound from range2

			range1.High = range2.High
			range1.Inclusion = (range1.Inclusion & datastore.LOW) |
				(range2.Inclusion & datastore.HIGH)
			range1.Selec2 = range2.Selec2
			changed = true
		} else {
			// Keep the lesser or unknown high bound from
			// range1 and range2

			high1 := range1.High.Value()
			high2 := range2.High.Value()

			if exact1 && (high1 == nil || high2 == nil) {
				exact1 = false
			}

			var use2, useBoth, useLeast bool
			if high1 != nil && high2 != nil {
				res := high1.Collate(high2)
				if res > 0 {
					use2 = true
				} else if res == 0 {
					useBoth = true
				}
			} else if high1 != nil {
				if high1.Type() > value.NULL {
					useLeast = true
				} else {
					use2 = true
				}
			} else if high2 != nil {
				if high2.Type() > value.NULL {
					useLeast = true
				}
			}
			if useBoth {
				range1.Inclusion = (range1.Inclusion & datastore.LOW) |
					(range1.Inclusion & range2.Inclusion & datastore.HIGH)
				changed = true
			} else if useLeast {
				exact1 = false
				range1.High = getLowHigh(range1.High, range2.High, false)
				range1.Inclusion = (range1.Inclusion & datastore.LOW) |
					((range1.Inclusion | range2.Inclusion) & datastore.HIGH)
				changed = true
			} else if use2 {
				range1.High = range2.High
				range1.Inclusion = (range1.Inclusion & datastore.LOW) |
					(range2.Inclusion & datastore.HIGH)
				range1.Selec2 = range2.Selec2
				changed = true
			}
		}
	}

	if changed {
		range1.SetCheckSpecialSpan()
		range1.InheritFlags(range2)
	}

	return exact1
}

func constrainEmptySpan(range1, range2 *plan.Range2) bool {
	// handle empty span for f1 >= 3 and f1 < 3, f1 < 3 and f1 >= 3

	if range1.High == nil || range2.Low == nil {
		return false
	}

	// range1 HIGH, range2 LOW are set, so it will not empty span
	if (range1.Inclusion&datastore.HIGH) != 0 && (range2.Inclusion&datastore.LOW) != 0 {
		return false
	}

	// range1 HIGH, range2 LOW are not set, so it will not empty span
	if (range1.Inclusion&datastore.HIGH) == 0 && (range2.Inclusion&datastore.LOW) == 0 {
		return false
	}

	high1 := range1.High.Value()
	low2 := range2.Low.Value()
	if low2 != nil && high1 != nil && high1.Equals(low2).Truth() {
		return true
	}
	return false
}

func streamline(cspans plan.Spans2) *TermSpans {
	switch len(cspans) {
	case 0:
		return _EMPTY_SPANS.Copy().(*TermSpans)
	case 1:
		if cspans[0].Empty() {
			return _EMPTY_SPANS.Copy().(*TermSpans)
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
		return _EMPTY_SPANS.Copy().(*TermSpans)
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
	} else if rg.EquivalentTo(_NOT_VALUED_SPAN.Ranges[0]) {
		rg.Flags |= plan.RANGE_NOT_VALUED_SPAN
	}
}

func isSpecialSargSpan(sspans SargSpans, flag uint32) bool {
	if tspans, ok := sspans.(*TermSpans); ok {
		return isSpecialSpan(tspans.spans, flag)
	}
	return false
}

func isSpecialSpan(spans plan.Spans2, flag uint32) bool {
	return len(spans) == 1 && len(spans[0].Ranges) == 1 && (spans[0].Ranges[0].Flags&flag) != 0

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

func getLowHigh(exp1, exp2 expression.Expression, low bool) expression.Expression {
	operands := make(expression.Expressions, 0, 4)
	operands = addLowHigh(operands, exp1, low)
	operands = addLowHigh(operands, exp2, low)
	if low {
		return expression.NewGreatest(operands...)
	}
	return expression.NewLeast(operands...)
}

func addLowHigh(operands expression.Expressions, exp expression.Expression, low bool) expression.Expressions {
	switch e := exp.(type) {
	case *expression.Greatest:
		if low {
			operands = append(operands, e.Operands()...)
		} else {
			operands = append(operands, exp)
		}
	case *expression.Least:
		if low {
			operands = append(operands, exp)
		} else {
			operands = append(operands, e.Operands()...)
		}
	case nil:
		// no-op
	default:
		operands = append(operands, exp)
	}
	return operands
}

var _STRING_SPAN_POOL = plan.NewStringSpanPool(1024)
