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
	CreateScan(index datastore.Index, term *algebra.KeyspaceTerm, distinct, overlap,
		array bool, limit expression.Expression, covers expression.Covers,
		filterCovers map[*expression.Cover]value.Value) plan.SecondaryScan

	Compose(prev SargSpans) SargSpans                       // Apply to previous composite keys
	ComposeTerm(next *TermSpans) SargSpans                  // Apply next composite keys
	Constrain(other SargSpans) SargSpans                    // Apply AND constraint
	ConstrainTerm(spans *TermSpans) SargSpans               // Apply AND constraint
	Streamline() SargSpans                                  // Dedup and discard empty spans
	Exact() bool                                            // Are all spans exact
	SetExact(exact bool)                                    // Set exact on spans
	SetExactForComposite(sargLength int) bool               // Set exact on spans
	MissingHigh() bool                                      // Missing high bound in any span
	CanUseIndexOrder() bool                                 // Can use index ORDER
	SkipsLeadingNulls() bool                                // For COUNT and MIN pushdown
	EquivalenceRangeAt(i int) (bool, expression.Expression) // For index ORDER on equivalence predicates
	Size() int                                              // Total number of spans
	Copy() SargSpans                                        // Deep copy
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

var _SELF_SPAN *plan.Span
var _SELF_SPANS *TermSpans

var _FULL_SPAN *plan.Span
var _FULL_SPANS *TermSpans

var _VALUED_SPAN *plan.Span
var _VALUED_SPANS *TermSpans

var _EMPTY_SPAN *plan.Span
var _EMPTY_SPANS *TermSpans

var _EXACT_FULL_SPAN *plan.Span
var _EXACT_FULL_SPANS *TermSpans

var _EXACT_VALUED_SPAN *plan.Span
var _EXACT_VALUED_SPANS *TermSpans

func init() {
	_SELF_SPAN = &plan.Span{}
	_SELF_SPAN.Range.Low = expression.Expressions{expression.TRUE_EXPR}
	_SELF_SPAN.Range.Inclusion = datastore.LOW
	_SELF_SPANS = NewTermSpans(_SELF_SPAN)

	_FULL_SPAN = &plan.Span{}
	_FULL_SPAN.Range.Low = expression.Expressions{expression.NULL_EXPR}
	_FULL_SPAN.Range.Inclusion = datastore.LOW
	_FULL_SPANS = NewTermSpans(_FULL_SPAN)

	_VALUED_SPAN = &plan.Span{}
	_VALUED_SPAN.Range.Low = expression.Expressions{expression.NULL_EXPR}
	_VALUED_SPAN.Range.Inclusion = datastore.NEITHER
	_VALUED_SPANS = NewTermSpans(_VALUED_SPAN)

	_EMPTY_SPAN = &plan.Span{}
	_EMPTY_SPAN.Range.High = expression.Expressions{expression.NULL_EXPR}
	_EMPTY_SPAN.Range.Inclusion = datastore.NEITHER
	_EMPTY_SPAN.Exact = true
	_EMPTY_SPANS = NewTermSpans(_EMPTY_SPAN)

	_EXACT_FULL_SPAN = _FULL_SPAN.Copy()
	_EXACT_FULL_SPAN.Exact = true
	_EXACT_FULL_SPANS = NewTermSpans(_EXACT_FULL_SPAN)

	_EXACT_VALUED_SPAN = _VALUED_SPAN.Copy()
	_EXACT_VALUED_SPAN.Exact = true
	_EXACT_VALUED_SPANS = NewTermSpans(_EXACT_VALUED_SPAN)
}
