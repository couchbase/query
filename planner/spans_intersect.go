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

type IntersectSpans struct {
	multiSpansBase
}

func NewIntersectSpans(spans ...SargSpans) *IntersectSpans {
	rv := &IntersectSpans{
		multiSpansBase{
			spans: spans,
		},
	}

	return rv
}

func (this *IntersectSpans) CreateScan(
	index datastore.Index, term *algebra.KeyspaceTerm, indexApiVersion int,
	reverse, distinct, overlap, array bool, offset, limit expression.Expression,
	projection *plan.IndexProjection, indexOrder plan.IndexKeyOrders,
	indexGroupAggs *plan.IndexGroupAggregates, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value, filter expression.Expression,
	cost, cardinality float64, size int64, frCost float64,
	baseKeyspace *base.BaseKeyspace, hasDeltaKeyspace, skipNewKeys, nested_loop bool) plan.SecondaryScan {

	if len(this.spans) == 1 {
		return this.spans[0].CreateScan(index, term, indexApiVersion, reverse, distinct,
			overlap, array, offset, limit, projection, indexOrder, indexGroupAggs,
			covers, filterCovers, filter, cost, cardinality, size, frCost,
			baseKeyspace, hasDeltaKeyspace, skipNewKeys, nested_loop)
	}

	scans := make([]plan.SecondaryScan, len(this.spans))
	for i, s := range this.spans {
		// No LIMIT pushdown
		scans[i] = s.CreateScan(index, term, indexApiVersion, reverse, distinct,
			false, array, nil, nil, projection, nil, indexGroupAggs,
			covers, filterCovers, filter, cost, cardinality, size, frCost,
			baseKeyspace, hasDeltaKeyspace, skipNewKeys, nested_loop)
	}

	limit = offsetPlusLimit(offset, limit)
	return plan.NewIntersectScan(limit, false, cost, cardinality, size, frCost, scans...)
}

func (this *IntersectSpans) Compose(prev SargSpans) SargSpans {
	this.compose(prev)
	return this
}

func (this *IntersectSpans) ComposeTerm(next *TermSpans) SargSpans {
	this.composeTerm(next)
	return this
}

func (this *IntersectSpans) Constrain(other SargSpans) SargSpans {
	this.constrain(other)
	return this
}

func (this *IntersectSpans) ConstrainTerm(spans *TermSpans) SargSpans {
	this.constrainTerm(spans)
	return this
}

func (this *IntersectSpans) Streamline() SargSpans {
	exactFull := false
	whole := false
	spans := _SPANS_POOL.Get()
	defer _SPANS_POOL.Put(spans)

	var sps []SargSpans
	for _, span := range this.spans {
		span = span.Streamline()

		switch span := span.(type) {
		case *IntersectSpans:
			sps = span.spans
		default:
			sps = []SargSpans{span}
		}

		for _, s := range sps {
			if s == _EMPTY_SPANS {
				return s
			} else if s == _WHOLE_SPANS {
				whole = true
			} else if s == _EXACT_FULL_SPANS {
				exactFull = true
			} else if s != _FULL_SPANS {
				spans = append(spans, s)
			}
		}
	}

	spans = dedupSpans(spans)

	switch len(spans) {
	case 0:
		if whole {
			return _WHOLE_SPANS
		} else if exactFull {
			return _EXACT_FULL_SPANS
		} else {
			return _FULL_SPANS
		}
	case 1:
		return spans[0]
	default:
		return NewIntersectSpans(spans...)
	}
}

func (this *IntersectSpans) CanUseIndexOrder(allowMultipleSpans bool) bool {
	return len(this.spans) == 1 && this.spans[0].CanUseIndexOrder(allowMultipleSpans)
}

func (this *IntersectSpans) CanPushDownOffset(index datastore.Index, overlap, array bool) bool {
	return len(this.spans) == 1 && this.spans[0].CanPushDownOffset(index, overlap, array)
}

func (this *IntersectSpans) CanHaveDuplicates(index datastore.Index, indexApiVersion int, overlap, array bool) bool {
	return len(this.spans) == 1 && this.spans[0].CanHaveDuplicates(index, indexApiVersion, overlap, array)
}

func (this *IntersectSpans) CanProduceUnknowns(pos int) bool {
	for _, span := range this.spans {
		if !span.CanProduceUnknowns(pos) {
			return false
		}
	}

	return true
}

func (this *IntersectSpans) SkipsLeadingNulls() bool {
	for _, span := range this.spans {
		if span.SkipsLeadingNulls() {
			return true
		}
	}

	return false
}

func (this *IntersectSpans) Size() int {
	size := 1
	for _, s := range this.spans {
		if sz := s.Size(); sz > 0 {
			size *= sz
		}
	}

	return size
}

func (this *IntersectSpans) Copy() SargSpans {
	return NewIntersectSpans(CopyAllSpans(this.spans)...)
}

func (this *IntersectSpans) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IntersectSpans) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{
		"#":     "IntersectSpans",
		"spans": this.spans,
	}

	return json.Marshal(r)
}
