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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

type SargSpans interface {
	fmt.Stringer
	json.Marshaler

	// Create index scan
	CreateScan(index datastore.Index, term *algebra.KeyspaceTerm, indexApiVersion int,
		reverse, distinct, overlap, array bool, offset, limit expression.Expression,
		projection *plan.IndexProjection, indexOrder plan.IndexKeyOrders,
		indexGroupAggs *plan.IndexGroupAggregates, covers expression.Covers,
		filterCovers map[*expression.Cover]value.Value, filter expression.Expression,
		cost, cardinality float64, size int64, frCost float64,
		baseKeyspace *base.BaseKeyspace, hasDeltaKeyspace, skipNewKeys, nested_loop bool) plan.SecondaryScan

	Compose(prev SargSpans) SargSpans              // Apply to previous composite keys
	ComposeTerm(next *TermSpans) SargSpans         // Apply next composite keys
	Constrain(other SargSpans) SargSpans           // Apply AND constraint
	ConstrainTerm(spans *TermSpans) SargSpans      // Apply AND constraint
	Streamline() SargSpans                         // Dedup and discard empty spans
	Exact() bool                                   // Are all spans exact
	ExactSpan1(nkeys int) bool                     // Are all spans exact - Api1
	SetExact(exact bool)                           // Set exact on spans
	HasStatic() bool                               // Has Static spans
	CanUseIndexOrder(allowMultipleSpans bool) bool // Can use index ORDER
	CanHaveDuplicates(index datastore.Index, indexApiVersion int,
		overlap, array bool) bool // Can have duplicates
	CanPushDownOffset(index datastore.Index, overlap, array bool) bool // Can offset pushdown index
	SkipsLeadingNulls() bool                                           // For COUNT and MIN pushdown
	EquivalenceRangeAt(i int) (bool, expression.Expression)            // For index ORDER on equivalence predicates
	CanProduceUnknowns(pos int) bool                                   // Index key pos can produce MISSING or NULL
	Size() int                                                         // Total number of spans
	Copy() SargSpans                                                   // Deep copy
}

func CopySpans(spans SargSpans) SargSpans {
	if spans == nil {
		return nil
	} else {
		return spans.Copy()
	}
}

func CopyAllSpans(spans []SargSpans) []SargSpans {
	spans2 := make([]SargSpans, len(spans))
	for i, s := range spans {
		spans2[i] = CopySpans(s)
	}

	return spans2
}

var _SELF_SPAN *plan.Span2
var _SELF_SPANS *TermSpans

var _FULL_SPAN *plan.Span2
var _FULL_SPANS *TermSpans

var _WHOLE_SPAN *plan.Span2
var _WHOLE_SPANS *TermSpans

var _VALUED_SPAN *plan.Span2
var _VALUED_SPANS *TermSpans

var _EMPTY_SPAN *plan.Span2
var _EMPTY_SPANS *TermSpans

var _MISSING_SPAN *plan.Span2
var _MISSING_SPANS *TermSpans

var _NULL_SPAN *plan.Span2
var _NULL_SPANS *TermSpans

var _NOT_VALUED_SPAN *plan.Span2
var _NOT_VALUED_SPANS *TermSpans

var _EXACT_FULL_SPAN *plan.Span2
var _EXACT_FULL_SPANS *TermSpans

var _EXACT_VALUED_SPAN *plan.Span2
var _EXACT_VALUED_SPANS *TermSpans

var _EXACT_SELF_SPAN *plan.Span2
var _EXACT_SELF_SPANS *TermSpans

func init() {
	var range2 *plan.Range2

	range2 = plan.NewRange2(expression.TRUE_EXPR, nil, datastore.LOW, OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL,
		plan.RANGE_SELF_SPAN)
	_SELF_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, false)
	_EXACT_SELF_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, true)
	_SELF_SPANS = NewTermSpans(_SELF_SPAN)
	_EXACT_SELF_SPANS = NewTermSpans(_EXACT_SELF_SPAN)

	range2 = plan.NewRange2(expression.NULL_EXPR, nil, datastore.LOW, OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL,
		plan.RANGE_FULL_SPAN)
	_FULL_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, false)
	_EXACT_FULL_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, true)
	_FULL_SPANS = NewTermSpans(_FULL_SPAN)
	_EXACT_FULL_SPANS = NewTermSpans(_EXACT_FULL_SPAN)

	range2 = plan.NewRange2(nil, nil, datastore.NEITHER, OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL, plan.RANGE_WHOLE_SPAN)
	_WHOLE_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, true)
	_WHOLE_SPANS = NewTermSpans(_WHOLE_SPAN)

	range2 = plan.NewRange2(expression.NULL_EXPR, nil, datastore.NEITHER, OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL,
		plan.RANGE_VALUED_SPAN)
	_VALUED_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, false)
	_EXACT_VALUED_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, true)
	_VALUED_SPANS = NewTermSpans(_VALUED_SPAN)
	_EXACT_VALUED_SPANS = NewTermSpans(_EXACT_VALUED_SPAN)

	range2 = plan.NewRange2(expression.NULL_EXPR, expression.NULL_EXPR, datastore.NEITHER, OPT_SELEC_NOT_AVAIL,
		OPT_SELEC_NOT_AVAIL, plan.RANGE_EMPTY_SPAN)
	_EMPTY_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, true)
	_EMPTY_SPANS = NewTermSpans(_EMPTY_SPAN)

	range2 = plan.NewRange2(expression.NULL_EXPR, expression.NULL_EXPR, datastore.BOTH, OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL,
		plan.RANGE_NULL_SPAN)
	_NULL_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, true)
	_NULL_SPANS = NewTermSpans(_NULL_SPAN)

	range2 = plan.NewRange2(nil, expression.NULL_EXPR, datastore.NEITHER, OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL,
		plan.RANGE_MISSING_SPAN)
	_MISSING_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, true)
	_MISSING_SPANS = NewTermSpans(_MISSING_SPAN)

	range2 = plan.NewRange2(nil, expression.NULL_EXPR, datastore.HIGH, OPT_SELEC_NOT_AVAIL, OPT_SELEC_NOT_AVAIL,
		plan.RANGE_NOT_VALUED_SPAN)
	_NOT_VALUED_SPAN = plan.NewStaticSpan2(nil, plan.Ranges2{range2}, true)
	_NOT_VALUED_SPANS = NewTermSpans(_NOT_VALUED_SPAN)
}
