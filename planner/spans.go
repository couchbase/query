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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type SargSpans interface {
	fmt.Stringer
	json.Marshaler

	// Create index scan
	CreateScan(index datastore.Index, term *algebra.KeyspaceTerm, indexApiVersion int, reverse, distinct, overlap,
		array bool, offset, limit expression.Expression, projection *plan.IndexProjection,
		indexOrder plan.IndexKeyOrders, indexGroupAggs *plan.IndexGroupAggregates, covers expression.Covers,
		filterCovers map[*expression.Cover]value.Value) plan.SecondaryScan

	Compose(prev SargSpans) SargSpans              // Apply to previous composite keys
	ComposeTerm(next *TermSpans) SargSpans         // Apply next composite keys
	Constrain(other SargSpans) SargSpans           // Apply AND constraint
	ConstrainTerm(spans *TermSpans) SargSpans      // Apply AND constraint
	Streamline() SargSpans                         // Dedup and discard empty spans
	Exact() bool                                   // Are all spans exact
	ExactSpan1(nkeys int) bool                     // Are all spans exact - Api1
	SetExact(exact bool)                           // Set exact on spans
	CanUseIndexOrder(allowMultipleSpans bool) bool // Can use index ORDER
	CanHaveDuplicates(index datastore.Index, indexApiVersion int,
		overlap, array bool) bool // Can have duplicates
	CanPushDownOffset(index datastore.Index, overlap, array bool) bool // Can offset pushdown index
	SkipsLeadingNulls() bool                                           // For COUNT and MIN pushdown
	EquivalenceRangeAt(i int) (bool, expression.Expression)            // For index ORDER on equivalence predicates
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

var _NULL_SPAN *plan.Span2
var _NULL_SPANS *TermSpans

var _EXACT_FULL_SPAN *plan.Span2
var _EXACT_FULL_SPANS *TermSpans

var _EXACT_VALUED_SPAN *plan.Span2
var _EXACT_VALUED_SPANS *TermSpans

func init() {
	var range2 *plan.Range2

	range2 = plan.NewRange2(expression.TRUE_EXPR, nil, datastore.LOW)
	_SELF_SPAN = plan.NewSpan2(nil, plan.Ranges2{range2}, false)
	_SELF_SPANS = NewTermSpans(_SELF_SPAN)

	range2 = plan.NewRange2(expression.NULL_EXPR, nil, datastore.LOW)
	_FULL_SPAN = plan.NewSpan2(nil, plan.Ranges2{range2}, false)
	_EXACT_FULL_SPAN = plan.NewSpan2(nil, plan.Ranges2{range2}, true)
	_FULL_SPANS = NewTermSpans(_FULL_SPAN)
	_EXACT_FULL_SPANS = NewTermSpans(_EXACT_FULL_SPAN)

	range2 = plan.NewRange2(nil, nil, datastore.NEITHER)
	_WHOLE_SPAN = plan.NewSpan2(nil, plan.Ranges2{range2}, true)
	_WHOLE_SPANS = NewTermSpans(_WHOLE_SPAN)

	range2 = plan.NewRange2(expression.NULL_EXPR, nil, datastore.NEITHER)
	_VALUED_SPAN = plan.NewSpan2(nil, plan.Ranges2{range2}, false)
	_EXACT_VALUED_SPAN = plan.NewSpan2(nil, plan.Ranges2{range2}, true)
	_VALUED_SPANS = NewTermSpans(_VALUED_SPAN)
	_EXACT_VALUED_SPANS = NewTermSpans(_EXACT_VALUED_SPAN)

	range2 = plan.NewRange2(expression.NULL_EXPR, expression.NULL_EXPR, datastore.NEITHER)
	_EMPTY_SPAN = plan.NewSpan2(nil, plan.Ranges2{range2}, true)
	_EMPTY_SPANS = NewTermSpans(_EMPTY_SPAN)

	range2 = plan.NewRange2(expression.NULL_EXPR, expression.NULL_EXPR, datastore.BOTH)
	_NULL_SPAN = plan.NewSpan2(nil, plan.Ranges2{range2}, true)
	_NULL_SPANS = NewTermSpans(_NULL_SPAN)
}
