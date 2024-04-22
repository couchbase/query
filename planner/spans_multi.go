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
	"github.com/couchbase/query/util"
)

// Shared base class
type multiSpansBase struct {
	spans []SargSpans
}

func (this *multiSpansBase) compose(prev SargSpans) {
	for i, s := range this.spans {
		this.spans[i] = s.Compose(prev)
	}
}

func (this *multiSpansBase) composeTerm(next *TermSpans) {
	for i, s := range this.spans {
		this.spans[i] = s.ComposeTerm(next)
	}
}

func (this *multiSpansBase) constrain(other SargSpans) {
	for i, s := range this.spans {
		this.spans[i] = s.Constrain(other)
	}
}

func (this *multiSpansBase) constrainTerm(spans *TermSpans) {
	for i, s := range this.spans {
		this.spans[i] = s.ConstrainTerm(spans)
	}
}

func (this *multiSpansBase) Exact() bool {
	for _, s := range this.spans {
		if !s.Exact() {
			return false
		}
	}

	return true
}

func (this *multiSpansBase) ExactSpan1(nkeys int) bool {
	for _, s := range this.spans {
		if !s.ExactSpan1(nkeys) {
			return false
		}
	}

	return true
}

func (this *multiSpansBase) SetExact(exact bool) {
	for _, s := range this.spans {
		s.SetExact(exact)
	}
}

func (this *multiSpansBase) HasStatic() bool {
	for _, s := range this.spans {
		if s.HasStatic() {
			return true
		}
	}
	return false
}

func (this *multiSpansBase) EquivalenceRangeAt(pos int) (eq bool, expr expression.Expression) {
	missing := false //To mark IS MISSING range

	for i, s := range this.spans {
		seq, sexpr := s.EquivalenceRangeAt(pos)
		if i == 0 && seq {
			missing = (sexpr == nil)
			expr = sexpr
		} else if !seq || !expression.Equivalent(expr, sexpr) {
			return false, nil
		}
	}

	return (expr != nil || missing), expr
}

func (this *multiSpansBase) SetArrayId(id int) {
	for _, span := range this.spans {
		span.SetArrayId(id)
	}
}

func (this *multiSpansBase) ArrayId() int {
	arrayId := 0
	for _, span := range this.spans {
		childId := span.ArrayId()
		if childId > 0 {
			if arrayId == 0 {
				arrayId = childId
			} else if arrayId != childId {
				return -1 // signal different arrayId found
			}
		} else if childId < 0 {
			return childId
		}
	}
	return arrayId
}

func dedupSpans(spans []SargSpans) []SargSpans {
	if len(spans) <= 1 {
		return spans
	}

	rv := make([]SargSpans, 0, len(spans))
	hash := _STRING_SPANS_POOL.Get()
	defer _STRING_SPANS_POOL.Put(hash)

	for _, span := range spans {
		s := span.String()
		if _, ok := hash[s]; !ok {
			hash[s] = true
			rv = append(rv, span)
		}
	}

	return rv
}

var _SPANS_POOL = NewSargSpansPool(16)
var _STRING_SPANS_POOL = util.NewStringBoolPool(16)
