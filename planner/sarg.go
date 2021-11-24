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

func SargFor(pred expression.Expression, entry *indexEntry, keys expression.Expressions, max int,
	isJoin, doSelec bool, baseKeyspace *base.BaseKeyspace, keyspaceNames map[string]string,
	advisorValidate bool, aliases map[string]bool, context *PrepareContext) (SargSpans, bool, error) {

	// Optimize top-level OR predicate
	if or, ok := pred.(*expression.Or); ok {
		return sargForOr(or, entry, keys, max, isJoin, doSelec, baseKeyspace, keyspaceNames,
			advisorValidate, aliases, context)
	}

	sargKeys := keys[0:max]

	// Get sarg spans for index sarg keys. The sarg spans are
	// truncated when they exceed the limit.
	sargSpans, exactSpan, err := getSargSpans(pred, sargKeys, isJoin, doSelec, baseKeyspace,
		keyspaceNames, advisorValidate, aliases, context)
	if sargSpans == nil || err != nil {
		return nil, exactSpan, err
	}

	return composeSargSpan(sargSpans, exactSpan)
}

func sargForOr(or *expression.Or, entry *indexEntry, keys expression.Expressions, max int,
	isJoin, doSelec bool, baseKeyspace *base.BaseKeyspace, keyspaceNames map[string]string,
	advisorValidate bool, aliases map[string]bool, context *PrepareContext) (SargSpans, bool, error) {

	exact := true
	spans := make([]SargSpans, len(or.Operands()))
	for i, c := range or.Operands() {
		_, max1, _, _ := SargableFor(c, keys, false, true, context, aliases) // Variable length sarging
		s, ex, err := SargFor(c, entry, keys, max1, isJoin, doSelec, baseKeyspace,
			keyspaceNames, advisorValidate, aliases, context)
		if err != nil {
			return nil, false, err
		}

		spans[i] = s
		exact = exact && ex

		if exact && (max1 < max) {
			// check for non-sargable key in predicate
			exprs, _, err := indexCoverExpressions(entry, keys[:max1], c, nil, baseKeyspace.Name(), context)
			if err != nil {
				return nil, false, err
			}
			implicitAny := implicitAnyCover(entry, true, context.FeatureControls())
			if !expression.IsCovered(c, baseKeyspace.Name(), exprs, implicitAny) {
				exact = false
			}
		}
	}

	var rv SargSpans = NewUnionSpans(spans...)
	return rv.Streamline(), exact, nil
}

func sargFor(pred, key expression.Expression, isJoin, doSelec bool, baseKeyspace *base.BaseKeyspace,
	keyspaceNames map[string]string, advisorValidate bool, aliases map[string]bool,
	context *PrepareContext) (SargSpans, error) {

	s := &sarg{key, baseKeyspace, keyspaceNames, isJoin, doSelec, advisorValidate, aliases, context}

	r, err := pred.Accept(s)
	if err != nil || r == nil {
		return nil, err
	}

	rs := r.(SargSpans)
	return rs, nil
}

func SargForFilters(filters base.Filters, keys expression.Expressions, max int, underHash, doSelec bool,
	baseKeyspace *base.BaseKeyspace, keyspaceNames map[string]string, advisorValidate bool,
	aliases map[string]bool, context *PrepareContext) (SargSpans, bool, error) {

	sargSpans := make([]SargSpans, max)
	exactSpan := true
	arrayKeySpans := make(map[int][]SargSpans)

	sargKeys := keys[0:max]

	for _, fl := range filters {
		if (baseKeyspace.OnclauseOnly() || baseKeyspace.IsOuter()) && !fl.IsOnclause() {
			// only ON-clause filter should be used
			continue
		}

		isJoin := fl.IsJoin() && !underHash
		flSargSpans, flExactSpan, err := getSargSpans(fl.FltrExpr(), sargKeys, isJoin,
			doSelec, baseKeyspace, keyspaceNames, advisorValidate, aliases, context)
		if err != nil {
			return nil, flExactSpan, err
		}

		exactSpan = exactSpan && flExactSpan

		for pos, sargKey := range sargKeys {
			isArray, _, _ := sargKey.IsArrayIndexKey()
			if flSargSpans[pos] == nil || flSargSpans[pos].Size() == 0 {
				if exactSpan && !isArray && fl.FltrExpr().DependsOn(sargKey) {
					exactSpan = false
				}
				continue
			} else if !isArray && flSargSpans[pos] == _EMPTY_SPANS {
				return _EMPTY_SPANS, true, nil
			}

			if isArray {
				if _, ok := arrayKeySpans[pos]; !ok {
					arrayKeySpans[pos] = make([]SargSpans, 0, len(filters))
				}
				arrayKeySpans[pos] = append(arrayKeySpans[pos], flSargSpans[pos])
			} else {
				if sargSpans[pos] == nil || sargSpans[pos].Size() == 0 {
					sargSpans[pos] = flSargSpans[pos]
				} else {
					sargSpans[pos] = sargSpans[pos].Constrain(flSargSpans[pos])
					if sargSpans[pos] == _EMPTY_SPANS {
						return _EMPTY_SPANS, true, nil
					}
				}
			}
		}
	}

	for pos, arrayKeySpan := range arrayKeySpans {
		sargSpans[pos] = addArrayKeys(arrayKeySpan)
	}

	if exactSpan {
		var hasSpan bool
		for _, s := range sargSpans {
			if s != nil {
				hasSpan = true
				break
			}
		}

		if !hasSpan {
			exactSpan = false
		}
	}

	return composeSargSpan(sargSpans, exactSpan)
}

/*
Compose SargSpan for a composite index
*/
func composeSargSpan(sargSpans []SargSpans, exactSpan bool) (SargSpans, bool, error) {
	// Truncate sarg spans when they exceed the limit
	size := 1
	n := 0
	for _, spans := range sargSpans {
		sz := 1
		if spans != nil {
			sz = spans.Size()
		}

		if sz == 0 ||
			(sz > 1 && size > 1 && sz*size > plan.FULL_SPAN_FANOUT) {
			exactSpan = false
			break
		}

		size *= sz
		n++
	}

	var ns SargSpans

	// Sarg composite indexes right to left
	for i := n - 1; i >= 0; i-- {
		rs := sargSpans[i]

		if rs == nil {
			rs = _WHOLE_SPANS.Copy()
		}
		if rs.Size() == 0 { // Reset
			ns = nil
			continue
		}

		// Start
		if ns == nil {
			ns = rs
			continue
		}

		ns = ns.Copy()
		ns = ns.Compose(rs)
		ns = ns.Streamline()

		if ns == _EMPTY_SPANS {
			return _EMPTY_SPANS, true, nil
		}
	}

	if ns == nil || ns.Size() == 0 {
		return _EMPTY_SPANS, true, nil
	}

	if ns.Exact() && !exactSpan {
		ns.SetExact(exactSpan)
	}

	return ns, exactSpan, nil
}

/*
Get sarg spans for index sarg keys.
*/
func getSargSpans(pred expression.Expression, sargKeys expression.Expressions, isJoin, doSelec bool,
	baseKeyspace *base.BaseKeyspace, keyspaceNames map[string]string, advisorValidate bool,
	aliases map[string]bool, context *PrepareContext) ([]SargSpans, bool, error) {

	// is the predicate simple?
	simple := true
	switch pred.(type) {
	case *expression.And, *expression.Or, *expression.Not:
		simple = false
	}

	n := len(sargKeys)

	exactSpan := true
	sargSpans := make([]SargSpans, n)

	// Sarg composite indexes right to left
	for i := n - 1; i >= 0; i-- {
		s := &sarg{sargKeys[i], baseKeyspace, keyspaceNames, isJoin, doSelec,
			advisorValidate, aliases, context}
		r, err := pred.Accept(s)
		if err != nil {
			return nil, false, err
		}

		if r != nil {
			rs := r.(SargSpans)
			rs = rs.Streamline()

			sargSpans[i] = rs

			if rs.Size() == 0 {
				exactSpan = false
				continue
			}

			// If one key span is EMPTY then whole index span can be EMPTY
			if rs == _EMPTY_SPANS {
				// make sure the returned slice is of length n since
				// the caller assumes that (it'll be streamlined later)
				for j := n - 1; j >= 0; j-- {
					sargSpans[j] = _EMPTY_SPANS
				}
				return sargSpans, true, nil
			}

			if simple {
				// if the same simple predicate can be used to sarg multiple
				// index keys, we can safely just use the exactSpan information
				// from this key and disregard that of the previous keys since
				// the index keys are walked backwards.
				// Specifically, if it generate an exact span for one of the
				// index key, we can set exactSpan to true even if it is not exact
				// for a different key (which appears after this index key)
				exactSpan = rs.Exact()
			} else {
				exactSpan = exactSpan && rs.Exact()
			}
		} else if exactSpan && pred.DependsOn(sargKeys[i]) {
			exactSpan = false
		}
	}

	return sargSpans, exactSpan, nil
}
