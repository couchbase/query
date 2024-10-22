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

func (this *sarg) visitDefault(pred expression.Expression) (SargSpans, error) {
	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	var spans SargSpans
	if pred.PropagatesNull() {
		spans = _VALUED_SPANS
	} else if pred.PropagatesMissing() {
		spans = _FULL_SPANS
	}

	if spans != nil && pred.DependsOn(this.key) {
		return spans, nil
	}

	return nil, nil
}

func getDependsSpans(pred expression.Expression) (SargSpans, error) {
	propMissing := pred.PropagatesMissing()
	propNull := pred.PropagatesNull()
	if propMissing && propNull {
		return _VALUED_SPANS, nil
	} else if propNull && !propMissing {
		return _FULL_SPANS, nil
	}
	return nil, nil
}
