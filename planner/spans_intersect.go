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

type IntersectSpans struct {
	multiSpansBase
}

func NewIntersectSpans(spans ...SargSpans) *IntersectSpans {
	rv := &IntersectSpans{
		multiSpansBase{
			spans: spans,
		},
	}

	return rv
}

func (this *IntersectSpans) CreateScan(
	index datastore.Index, term *algebra.KeyspaceTerm, reverse, distinct, ordered, overlap,
	array bool, offset, limit expression.Expression, projection *plan.IndexProjection, covers expression.Covers,
	filterCovers map[*expression.Cover]value.Value) plan.SecondaryScan {

	if len(this.spans) == 1 {
		return this.spans[0].CreateScan(index, term, reverse, distinct, ordered, overlap, array, offset, limit, projection, covers, filterCovers)
	}

	scans := make([]plan.SecondaryScan, len(this.spans))
	for i, s := range this.spans {
		// No LIMIT pushdown
		scans[i] = s.CreateScan(index, term, reverse, distinct, false, false, true, nil, nil, projection, covers, filterCovers)
	}

	limit = offsetPlusLimit(offset, limit)
	return plan.NewIntersectScan(limit, scans...)
}

func (this *IntersectSpans) Compose(prev SargSpans) SargSpans {
	this.compose(prev)
	return this
}

func (this *IntersectSpans) ComposeTerm(next *TermSpans) SargSpans {
	this.composeTerm(next)
	return this
}

func (this *IntersectSpans) Constrain(other SargSpans) SargSpans {
	this.constrain(other)
	return this
}

func (this *IntersectSpans) ConstrainTerm(spans *TermSpans) SargSpans {
	this.constrainTerm(spans)
	return this
}

func (this *IntersectSpans) Streamline() SargSpans {
	exact := false
	spans := _SPANS_POOL.Get()
	defer _SPANS_POOL.Put(spans)

	var sps []SargSpans
	for _, span := range this.spans {
		span = span.Streamline()

		switch span := span.(type) {
		case *IntersectSpans:
			sps = span.spans
		default:
			sps = []SargSpans{span}
		}

		for _, s := range sps {
			if s == _EMPTY_SPANS {
				return s
			} else if s == _EXACT_FULL_SPANS {
				exact = true
			} else if s != _FULL_SPANS {
				spans = append(spans, s)
			}
		}
	}

	spans = dedupSpans(spans)

	switch len(spans) {
	case 0:
		if exact {
			return _EXACT_FULL_SPANS
		} else {
			return _FULL_SPANS
		}
	case 1:
		return spans[0]
	default:
		return NewIntersectSpans(spans...)
	}
}

func (this *IntersectSpans) CanUseIndexOrder() bool {
	return len(this.spans) == 1 && this.spans[0].CanUseIndexOrder()
}

func (this *IntersectSpans) CanPushDownOffset(index datastore.Index, overlap, array bool) bool {
	return len(this.spans) == 1 && this.spans[0].CanPushDownOffset(index, overlap, array)
}

func (this *IntersectSpans) CanHaveDuplicates(index datastore.Index, overlap, array bool) bool {
	return len(this.spans) == 1 && this.spans[0].CanHaveDuplicates(index, overlap, array)
}

func (this *IntersectSpans) SkipsLeadingNulls() bool {
	for _, span := range this.spans {
		if span.SkipsLeadingNulls() {
			return true
		}
	}

	return false
}

func (this *IntersectSpans) Size() int {
	size := 1
	for _, s := range this.spans {
		if sz := s.Size(); sz > 0 {
			size *= sz
		}
	}

	return size
}

func (this *IntersectSpans) Copy() SargSpans {
	return NewIntersectSpans(CopyAllSpans(this.spans)...)
}

func (this *IntersectSpans) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this *IntersectSpans) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{
		"#":     "IntersectSpans",
		"spans": this.spans,
	}

	return json.Marshal(r)
}
