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
	"github.com/couchbase/query/expression"
)

func SargFor(pred expression.Expression, keys expression.Expressions, min int, isJoin bool, keyspaceName string) (
	SargSpans, bool, error) {

	// Optimize top-level OR predicate
	if or, ok := pred.(*expression.Or); ok {
		return sargForOr(or, keys, min, isJoin, keyspaceName)
	}

	sargKeys := keys[0:min]

	// Get sarg spans for index sarg keys. The sarg spans are
	// truncated when they exceed the limit.
	sargSpans, exactSpan, err := getSargSpans(pred, sargKeys, isJoin, keyspaceName)
	if sargSpans == nil || err != nil {
		return nil, exactSpan, err
	}

	return composeSargSpan(sargSpans, exactSpan)
}

func sargForOr(or *expression.Or, keys expression.Expressions, min int, isJoin bool, keyspaceName string) (
	SargSpans, bool, error) {

	exact := true
	spans := make([]SargSpans, len(or.Operands()))
	for i, c := range or.Operands() {
		min, _ = SargableFor(c, keys) // Variable length sarging
		s, ex, err := SargFor(c, keys, min, isJoin, keyspaceName)
		if err != nil {
			return nil, false, err
		}

		spans[i] = s
		exact = exact && ex
	}

	var rv SargSpans = NewUnionSpans(spans...)
	return rv.Streamline(), exact, nil
}

func sargFor(pred, key expression.Expression, isJoin bool, keyspaceName string) (SargSpans, error) {
	s := &sarg{key, keyspaceName, isJoin}

	r, err := pred.Accept(s)
	if err != nil || r == nil {
		return nil, err
	}

	rs := r.(SargSpans)
	return rs, nil
}

func SargForFilters(filters Filters, keys expression.Expressions, min int, keyspaceName string) (
	SargSpans, bool, error) {

	sargSpans := make([]SargSpans, min)
	exactSpan := true
	arrayKeySpans := make(map[int][]SargSpans)

	sargKeys := keys[0:min]

	for _, fl := range filters {
		flSargSpans, flExactSpan, pos, err := getSargSpan(fl.fltrExpr, sargKeys, fl.isJoin, keyspaceName)
		if err != nil {
			return nil, flExactSpan, err
		}

		isArray, _ := sargKeys[pos].IsArrayIndexKey()

		if flSargSpans == nil || flSargSpans.Size() == 0 {
			if !isArray {
				for _, sargKey := range sargKeys {
					if fl.fltrExpr.DependsOn(sargKey) {
						exactSpan = false
						break
					}
				}
			}
			continue
		}

		if isArray {
			if arrayKeySpan, ok := arrayKeySpans[pos]; ok {
				arrayKeySpan = append(arrayKeySpan, flSargSpans)
				arrayKeySpans[pos] = arrayKeySpan
			} else {
				arrayKeySpans[pos] = make([]SargSpans, 0, len(filters))
				arrayKeySpans[pos] = append(arrayKeySpans[pos], flSargSpans)
			}
		} else {
			if flSargSpans == _EMPTY_SPANS {
				return _EMPTY_SPANS, true, nil
			}

			if sargSpans[pos] == nil || sargSpans[pos].Size() == 0 {
				sargSpans[pos] = flSargSpans
			} else {
				sargSpans[pos] = sargSpans[pos].Constrain(flSargSpans)
				if sargSpans[pos] == _EMPTY_SPANS {
					return _EMPTY_SPANS, true, nil
				}
			}
		}

		exactSpan = exactSpan && flExactSpan
	}

	for pos, arrayKeySpan := range arrayKeySpans {
		sargSpans[pos] = addArrayKeys(arrayKeySpan)
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
		if spans == nil {
			break
		}

		sz := spans.Size()

		if sz == 0 ||
			(sz > 1 && size > 1 && sz*size > _FULL_SPAN_FANOUT) {
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

		// Reset
		if rs == nil || rs.Size() == 0 {
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
func getSargSpans(pred expression.Expression, sargKeys expression.Expressions, isJoin bool, keyspaceName string) (
	[]SargSpans, bool, error) {

	n := len(sargKeys)

	exactSpan := true
	sargSpans := make([]SargSpans, n)

	// Sarg composite indexes right to left
	for i := n - 1; i >= 0; i-- {
		s := &sarg{sargKeys[i], keyspaceName, isJoin}
		r, err := pred.Accept(s)
		if err != nil || r == nil {
			return nil, false, err
		}

		rs := r.(SargSpans)
		rs = rs.Streamline()

		sargSpans[i] = rs

		if rs.Size() == 0 {
			exactSpan = false
			continue
		}

		// If one key span is EMPTY then whole index span can be EMPTY
		if rs == _EMPTY_SPANS {
			return []SargSpans{_EMPTY_SPANS}, true, nil
		}

		exactSpan = exactSpan && rs.Exact()
	}

	return sargSpans, exactSpan, nil
}

/*
Get sarg span for one of the index sarg keys. Individual filters are passed in
*/
func getSargSpan(pred expression.Expression, sargKeys expression.Expressions, isJoin bool, keyspaceName string) (
	SargSpans, bool, int, error) {

	n := len(sargKeys)

	exactSpan := true

	// Sarg composite indexes left to right
	for i := 0; i < n; i++ {
		s := &sarg{sargKeys[i], keyspaceName, isJoin}
		r, err := pred.Accept(s)
		if err != nil {
			return nil, false, 0, err
		}

		if r == nil {
			// not for this key
			continue
		}

		rs := r.(SargSpans)
		rs = rs.Streamline()

		if rs.Size() == 0 {
			exactSpan = false
		}

		exactSpan = exactSpan && rs.Exact()

		return rs, exactSpan, i, nil
	}

	return nil, false, 0, nil
}

const _FULL_SPAN_FANOUT = 8192
