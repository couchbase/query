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

func (this *sargable) VisitAnd(pred *expression.And) (interface{}, error) {
	if SubsetOf(pred, this.key) {
		return true, nil
	}

	keys := expression.Expressions{this.key}
	for _, child := range pred.Operands() {
		if min, _, _, _ := SargableFor(child, keys, this.missing, this.gsi); min > 0 {
			return true, nil
		}
	}

	return false, nil
}
