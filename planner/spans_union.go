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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type UnionSpans struct {
	multiSpansBase
}

func NewUnionSpans(spans ...SargSpans) *UnionSpans {
	rv := &UnionSpans{
		multiSpansBase{
			spans: spans,
		},
	}

	return rv
}

func (this *UnionSpans) CreateScan(
	index datastore.Index, term *algebra.KeyspaceTerm, reverse, distinct, ordered, overlap,
	array bool, offset, limit expression.Expression, projection *plan.IndexProjection, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value) plan.SecondaryScan {

	if len(this.spans) == 1 {
		return this.spans[0].CreateScan(index, term, reverse, distinct, ordered, overlap, array, offset, limit, projection, covers, filterCovers)
	}

	lim := offsetPlusLimit(offset, limit)
	scans := make([]plan.SecondaryScan, len(this.spans))
	for i, s := range this.spans {
		scans[i] = s.CreateScan(index, term, reverse, distinct, ordered, overlap, array, nil, lim, projection, covers, filterCovers)
	}

	return plan.NewUnionScan(limit, offset, scans...)
}

func (this *UnionSpans) Compose(prev SargSpans) SargSpans {
	prevs, ok := prev.(*UnionSpans)
	if !ok || len(prevs.spans) != len(this.spans) {
		prev = prev.Copy()
		prev.SetExact(false)
		return prev
	}

	for i, s := range this.spans {
		p := prevs.spans[i]
		if s != nil && p != nil {
			this.spans[i] = s.Compose(p)
		} else {
			this.spans[i] = p
		}
	}

	return this
}

func (this *UnionSpans) ComposeTerm(next *TermSpans) SargSpans {
	this.composeTerm(next)
	return this
}

func (this *UnionSpans) Constrain(other SargSpans) SargSpans {
	this.constrain(other)
	return this
}

func (this *UnionSpans) ConstrainTerm(spans *TermSpans) SargSpans {
	this.constrainTerm(spans)
	return this
}

func (this *UnionSpans) Streamline() SargSpans {
	full := false
	termSpans := make([]*TermSpans, 0, len(this.spans))
	spans := _SPANS_POOL.Get()
	defer _SPANS_POOL.Put(spans)

	var sps []SargSpans
	for _, span := range this.spans {

		span = span.Streamline()

		switch span := span.(type) {
		case *UnionSpans:
			sps = span.spans
		default:
			sps = []SargSpans{span}
		}

		for _, s := range sps {
			if s == _EXACT_FULL_SPANS {
				return s
			} else if s == _FULL_SPANS {
				full = true
			} else if s == _EMPTY_SPANS {
				continue
			} else if term, ok := s.(*TermSpans); ok {
				termSpans = append(termSpans, term)
			} else {
				spans = append(spans, s)
			}
		}
	}

	if full {
		return _FULL_SPANS
	}

	// Combine TermSpans
	switch len(termSpans) {
	case 0:
		// Do nothing
	case 1:
		spans = append(spans, termSpans[0])
	default:
		terms := make(plan.Spans2, 0, this.Size())
		for _, t := range termSpans {
			terms = append(terms, t.spans...)
		}

		ts := NewTermSpans(terms...).Streamline()
		spans = append(spans, ts)
	}

	spans = dedupSpans(spans)

	switch len(spans) {
	case 0:
		return _EMPTY_SPANS
	case 1:
		return spans[0]
	default:
		return NewUnionSpans(spans...)
	}
}

func (this *UnionSpans) CanUseIndexOrder() bool {
	return len(this.spans) == 1 && this.spans[0].CanUseIndexOrder()
}

func (this *UnionSpans) CanPushDownOffset(index datastore.Index, overlap, array bool) bool {
	for _, span := range this.spans {
		if !span.CanPushDownOffset(index, overlap, array) {
			return false
		}
	}

	return true
}

func (this *UnionSpans) CanHaveDuplicates(index datastore.Index, overlap, array bool) bool {
	for _, span := range this.spans {
		if !span.CanHaveDuplicates(index, overlap, array) {
			return false
		}
	}

	return true
}

func (this *UnionSpans) SkipsLeadingNulls() bool {
	for _, span := range this.spans {
		if !span.SkipsLeadingNulls() {
			return false
		}
	}

	return true
}

func (this *UnionSpans) Size() int {
	size := 0
	for _, s := range this.spans {
		size += s.Size()
	}

	return size
}

func (this *UnionSpans) Copy() SargSpans {
	return NewUnionSpans(CopyAllSpans(this.spans)...)
}

func (this *UnionSpans) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *UnionSpans) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{
		"#":     "UnionSpans",
		"spans": this.spans,
	}

	return json.Marshal(r)
}
