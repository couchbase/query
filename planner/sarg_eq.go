//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
	base "github.com/couchbase/query/plannerbase"
)

func (this *sarg) VisitEq(pred *expression.Eq) (interface{}, error) {
	if base.SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	var expr expression.Expression

	if pred.First().EquivalentTo(this.key) {
		expr = this.getSarg(pred.Second())
	} else if pred.Second().EquivalentTo(this.key) {
		expr = this.getSarg(pred.First())
	} else if pred.DependsOn(this.key) {
		return getDependsSpans(pred)
	} else {
		return nil, nil
	}

	if expr == nil {
		return _VALUED_SPANS, nil
	}

	selec := this.getSelec(pred)
	range2 := plan.NewRange2(expr, expr, datastore.BOTH, selec, OPT_SELEC_NOT_AVAIL, 0)
	span := plan.NewSpan2(nil, plan.Ranges2{range2}, true)
	return NewTermSpans(span), nil
}
