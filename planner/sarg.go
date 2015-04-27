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
)

func SargFor(pred expression.Expression, exprs expression.Expressions) (Spans, error) {
	n := SargableFor(pred, exprs)
	s := newSarg(pred)
	s.SetMissingHigh(n < len(exprs))
	var ns Spans

	// Sarg compositive indexes right to left
	for i := n - 1; i >= 0; i-- {
		r, err := exprs[i].Accept(s)
		if err != nil || r == nil {
			return nil, err
		}

		rs := r.(Spans)
		if len(rs) == 0 {
			// Should not reach here
			return nil, nil
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
		sp := make(Spans, 0, len(rs)*len(ns))
	prevs:
		for _, prev := range rs {
			// Full span subsumes others
			if len(prev.Range.Low) == 0 && len(prev.Range.High) == 0 {
				sp = append(sp, prev)
				continue
			}

			for _, next := range ns {
				// Full span subsumes others
				if len(next.Range.Low) == 0 && len(next.Range.High) == 0 {
					sp = append(sp, prev)
					continue prevs
				}
			}

			for j, next := range ns {
				pre := prev
				if j < len(ns)-1 {
					pre = pre.Copy()
				}

				if len(pre.Range.Low) > 0 && len(next.Range.Low) > 0 {
					pre.Range.Low = append(pre.Range.Low, next.Range.Low...)
					pre.Range.Inclusion = (datastore.LOW & pre.Range.Inclusion & next.Range.Inclusion) |
						(datastore.HIGH & pre.Range.Inclusion)
				}

				if len(pre.Range.High) > 0 && len(next.Range.High) > 0 {
					pre.Range.High = append(pre.Range.High, next.Range.High...)
					pre.Range.Inclusion = (datastore.HIGH & pre.Range.Inclusion & next.Range.Inclusion) |
						(datastore.LOW & pre.Range.Inclusion)
				}

				sp = append(sp, pre)
			}
		}

		ns = sp
	}

	return ns, nil
}

func sargFor(pred, expr expression.Expression, missingHigh bool) (Spans, error) {
	s := newSarg(pred)
	s.SetMissingHigh(missingHigh)

	r, err := expr.Accept(s)
	if err != nil || r == nil {
		return nil, err
	}

	rs := r.(Spans)
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
