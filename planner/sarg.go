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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
)

func SargFor(pred expression.Expression, sargKeys expression.Expressions, total int) (
	plan.Spans, bool, error) {

	n := len(sargKeys)
	s := newSarg(pred)
	s.SetMissingHigh(n < total)
	var ns plan.Spans

	// Sarg compositive indexes right to left
	exactSpan := true
keys:
	for i := n - 1; i >= 0; i-- {
		r, err := sargKeys[i].Accept(s)
		if err != nil || r == nil {
			return nil, false, err
		}

		rs := r.(plan.Spans)
		if len(rs) == 0 {
			ns = nil
			exactSpan = false
			continue
		} else if exactSpan {
			for _, prev := range rs {
				if !prev.Exact {
					exactSpan = false
					break
				}
			}
		}

		// Notify prev key that this key is missing a high bound
		if i > 0 {
			s.SetMissingHigh(false)
			for _, prev := range rs {
				if len(prev.Range.High) == 0 {
					s.SetMissingHigh(true)
					break
				}
			}
		}

		if ns == nil {
			// First iteration
			ns = rs
			continue
		}

		// Cross product of prev and next spans
		sp := make(plan.Spans, 0, len(rs)*len(ns))

		for _, prev := range rs {
			// Full span subsumes others
			if prev == _FULL_SPANS[0] || prev == _EXACT_FULL_SPANS[0] {
				exactSpan = false
				sp = append(sp, prev)
				ns = sp
				continue keys
			}
		}

	prevs:
		for _, prev := range rs {
			if len(prev.Range.Low) == 0 && len(prev.Range.High) == 0 {
				exactSpan = false
				sp = append(sp, prev)
				continue
			}

			// Limit fan-out
			if len(rs) > 1 && len(rs)*len(ns) > _FULL_SPAN_FANOUT {
				exactSpan = false
				sp = append(sp, prev)
				continue
			}

			for _, next := range ns {
				// Full span subsumes others
				if next == _FULL_SPANS[0] || next == _EXACT_FULL_SPANS[0] ||
					(len(next.Range.Low) == 0 && len(next.Range.High) == 0) {
					exactSpan = false
					sp = append(sp, prev)
					continue prevs
				}
			}

			pn := make(plan.Spans, 0, len(ns))
			for _, next := range ns {
				add := false
				pre := prev.Copy()

				if len(pre.Range.Low) > 0 && len(next.Range.Low) > 0 {
					pre.Range.Low = append(pre.Range.Low, next.Range.Low...)

					pre.Range.Inclusion = (datastore.LOW & pre.Range.Inclusion & next.Range.Inclusion) |
						(datastore.HIGH & pre.Range.Inclusion)
					add = true
				} else if len(pre.Range.Low) > 0 || len(next.Range.Low) > 0 {
					exactSpan = false
				}

				if len(pre.Range.High) > 0 && len(next.Range.High) > 0 {
					pre.Range.High = append(pre.Range.High, next.Range.High...)
					pre.Range.Inclusion = (datastore.HIGH & pre.Range.Inclusion & next.Range.Inclusion) |
						(datastore.LOW & pre.Range.Inclusion)
					add = true
				} else if len(pre.Range.High) > 0 || len(next.Range.High) > 0 {
					// f1 >=3 and f2 = 2 become span of {[3, 2] [] 1}, high of f2 is missing
					exactSpan = false
				}

				if add {
					pn = append(pn, pre)
				} else {
					exactSpan = false
					break
				}
			}

			if len(pn) == len(ns) {
				sp = append(sp, pn...)
			} else {
				exactSpan = false
				sp = append(sp, prev)
			}
		}

		ns = sp
	}

	if len(ns) == 0 {
		return _FULL_SPANS, false, nil
	}

	if exactSpan && len(sargKeys) > 1 {
		exactSpan = exactSpansForCompositeKeys(ns, sargKeys)
	}

	return ns, exactSpan, nil
}

func exactSpansForCompositeKeys(ns plan.Spans, sargKeys expression.Expressions) bool {

	for _, prev := range ns {
		if len(prev.Range.Low) != len(sargKeys) {
			return false
		}

		// trailing key is > or >=
		if len(prev.Range.High) < len(sargKeys)-1 {
			return false
		}

		// Except last key all leading keys needs to be EQ
		for i := 0; i < len(sargKeys)-1; i++ {
			low := prev.Range.Low[i].Value()
			high := prev.Range.High[i].Value()
			// Successor present in high Equals returns false
			if low == nil || high == nil || !low.Equals(high).Truth() {
				return false
			}
		}
	}
	return true
}

func sargFor(pred, expr expression.Expression, missingHigh bool) (plan.Spans, error) {
	s := newSarg(pred)
	s.SetMissingHigh(missingHigh)

	r, err := expr.Accept(s)
	if err != nil || r == nil {
		return nil, err
	}

	rs := r.(plan.Spans)
	return rs, nil
}

func newSarg(pred expression.Expression) sarg {
	s, _ := pred.Accept(_SARG_FACTORY)
	return s.(sarg)
}

type sarg interface {
	expression.Visitor
	SetMissingHigh(bool)
	MissingHigh() bool
}

const _FULL_SPAN_FANOUT = 8192
