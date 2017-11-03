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

func (this *sarg) VisitLT(pred *expression.LT) (interface{}, error) {
	if SubsetOf(pred, this.key) {
		return _SELF_SPANS, nil
	}

	var expr expression.Expression
	range2 := &plan.Range2{}

	if pred.First().EquivalentTo(this.key) {
		expr = this.getSarg(pred.Second())
		range2.Low = expression.NULL_EXPR
		range2.High = expr
	} else if pred.Second().EquivalentTo(this.key) {
		expr = this.getSarg(pred.First())
		range2.Low = expr
	} else if pred.DependsOn(this.key) {
		return _VALUED_SPANS, nil
	} else {
		return nil, nil
	}

	if expr == nil {
		return _VALUED_SPANS, nil
	}
	range2.Inclusion = datastore.NEITHER
	span := plan.NewSpan2(nil, plan.Ranges2{range2}, true)
	return NewTermSpans(span), nil
}
