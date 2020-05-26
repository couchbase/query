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
		return _VALUED_SPANS, nil
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
