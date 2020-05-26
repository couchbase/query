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
	"github.com/couchbase/query/expression"
)

func (this *sarg) VisitIsNotMissing(pred *expression.IsNotMissing) (interface{}, error) {
	if SubsetOf(pred, this.key) {
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
	if SubsetOf(pred, this.key) {
		if expression.Equivalent(pred, this.key) {
			return _EXACT_SELF_SPANS, nil
		}
		return _SELF_SPANS, nil
	}

	if pred.Operand().EquivalentTo(this.key) {
		return _MISSING_SPANS, nil
	}

	return nil, nil
}
