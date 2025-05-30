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
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/util"
)

func (this *sarg) VisitOr(pred *expression.Or) (interface{}, error) {
	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	spans := make([]SargSpans, 0, len(pred.Operands()))
	emptySpan := false
	missingSpan := false
	nullSpan := false
	fullSpan := false
	exactFullSpan := false
	valuedSpan := false
	exactValuedSpan := false
	size := 0

	for _, child := range pred.Operands() {
		cspans, _, err := sargFor(child, this.index, this.key, this.isJoin, this.doSelec,
			this.baseKeyspace, this.keyspaceNames, this.advisorValidate, this.isMissing,
			this.isArray, this.isVector, this.isInclude, this.keyPos, this.aliases, this.context)
		if err != nil {
			return nil, err
		}

		if cspans == nil || cspans.Size() == 0 {
			return cspans, nil
		} else if cspans == _WHOLE_SPANS {
			return cspans, nil
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

		size += cspans.Size()
		if size > util.FullSpanFanout(this.isInclude) {
			fullSpan = true
			continue
		}

		spans = append(spans, cspans)
	}

	if (missingSpan && exactFullSpan) || (missingSpan && nullSpan && exactValuedSpan) {
		return _WHOLE_SPANS, nil
	} else if (missingSpan && fullSpan) || (missingSpan && nullSpan && valuedSpan) {
		s := _WHOLE_SPANS.Copy()
		s.SetExact(false)
		return s, nil
	} else if (nullSpan && exactValuedSpan) || exactFullSpan {
		return _EXACT_FULL_SPANS, nil
	} else if (nullSpan && valuedSpan) || fullSpan {
		return _FULL_SPANS, nil
	} else if emptySpan && len(spans) == 0 {
		return _EMPTY_SPANS, nil
	}

	rv := NewUnionSpans(spans...)
	return rv.Streamline(), nil
}
