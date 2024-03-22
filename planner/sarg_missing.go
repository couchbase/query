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
)

func (this *sarg) VisitIsNotMissing(pred *expression.IsNotMissing) (interface{}, error) {
	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	if pred.Operand().EquivalentTo(this.key) {
		return _EXACT_FULL_SPANS, nil
	}

	return nil, nil
}

func (this *sarg) VisitIsMissing(pred *expression.IsMissing) (interface{}, error) {
	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	if pred.Operand().EquivalentTo(this.key) && this.isMissing {
		// MB-38287
		// For array index key requires whole scan because indexer doesn't have info.
		if !this.isArray {
			return _MISSING_SPANS, nil
		}
		s := _WHOLE_SPANS.Copy()
		s.SetExact(false)
		return s, nil
	}

	return nil, nil
}
