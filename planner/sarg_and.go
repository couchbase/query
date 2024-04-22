//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func (this *sarg) VisitAnd(pred *expression.And) (rv interface{}, err error) {
	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	// MB-21720. Handle array index keys differently.
	if isArray, _, _ := this.key.IsArrayIndexKey(); isArray {
		return this.visitAndArrayKey(pred, this.key)
	}

	var spans, s SargSpans
	var exact bool
	exactSpans := true

	for _, op := range pred.Operands() {
		s, exact, err = sargFor(op, this.key, this.isJoin, this.doSelec, this.baseKeyspace,
			this.keyspaceNames, this.advisorValidate, this.isMissing, this.isArray,
			this.aliases, this.context)
		if err != nil {
			return nil, err
		}

		if s == nil || s.Size() == 0 {
			if !exact {
				exactSpans = false
			}

			continue
		}

		if s == _EMPTY_SPANS {
			return _EMPTY_SPANS, nil
		}

		if spans == nil || spans.Size() == 0 {
			spans = s
		} else {
			spans = spans.Constrain(s)
			if spans == _EMPTY_SPANS {
				return _EMPTY_SPANS, nil
			}
		}
	}

	if !exactSpans && spans != nil && spans.Exact() {
		spans = spans.Copy()
		spans.SetExact(false)
	}

	return spans, nil
}

// MB-21720. Handle array index keys differently.
func (this *sarg) visitAndArrayKey(pred *expression.And, key expression.Expression) (SargSpans, error) {
	keySpans := make([]SargSpans, 0, len(pred.Operands()))

	for _, child := range pred.Operands() {
		cspans, _, err := sargFor(child, key, this.isJoin, this.doSelec, this.baseKeyspace,
			this.keyspaceNames, this.advisorValidate, this.isMissing, this.isArray,
			this.aliases, this.context)
		if err != nil {
			return nil, err
		}

		if cspans == nil || cspans.Size() == 0 {
			continue
		}
		keySpans = append(keySpans, cspans)
	}

	return addArrayKeys(keySpans), nil
}

func addArrayKeys(keySpans []SargSpans) SargSpans {

	spans := make([]SargSpans, 0, len(keySpans))
	emptySpan := false
	missingSpan := false
	nullSpan := false
	fullSpan := false
	exactFullSpan := false
	valuedSpan := false
	exactValuedSpan := false
	size := 1

	for _, cspans := range keySpans {
		if cspans == _WHOLE_SPANS {
			return _WHOLE_SPANS
		} else if cspans == _EXACT_FULL_SPANS {
			exactFullSpan = true
		} else if cspans == _FULL_SPANS {
			fullSpan = true
		} else if cspans == _EXACT_VALUED_SPANS {
			exactValuedSpan = true
		} else if cspans == _VALUED_SPANS {
			valuedSpan = true
		} else if cspans == _NULL_SPANS {
			nullSpan = true
		} else if cspans == _MISSING_SPANS {
			missingSpan = true
		} else if cspans == _NOT_VALUED_SPANS {
			missingSpan = true
			nullSpan = true
		} else if cspans == _EMPTY_SPANS {
			emptySpan = true
			continue
		}

		size *= cspans.Size()
		if size > plan.FULL_SPAN_FANOUT {
			fullSpan = true
			continue
		}

		spans = append(spans, cspans)
	}

	if (missingSpan && exactFullSpan) || (missingSpan && nullSpan && exactValuedSpan) {
		return _WHOLE_SPANS
	} else if (missingSpan && fullSpan) || (missingSpan && nullSpan && valuedSpan) {
		s := _WHOLE_SPANS.Copy()
		s.SetExact(false)
		return s
	} else if (nullSpan && exactValuedSpan) || exactFullSpan {
		return _EXACT_FULL_SPANS
	} else if (nullSpan && valuedSpan) || fullSpan {
		return _FULL_SPANS
	} else if emptySpan && len(spans) == 0 {
		return _EMPTY_SPANS
	} else if spans == nil || len(spans) == 0 {
		return nil
	}

	rv := NewIntersectSpans(spans...)
	return rv.Streamline()
}
