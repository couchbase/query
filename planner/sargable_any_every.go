//  Copyright (c) 2016 Couchbase, Inc.
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

type sargableAnyEvery struct {
	predicate
}

func newSargableAnyEvery(pred *expression.AnyEvery) *sargableAnyEvery {
	rv := &sargableAnyEvery{}
	rv.test = func(expr2 expression.Expression) (bool, error) {
		if defaultSargable(pred, expr2) {
			return true, nil
		}

		all, ok := expr2.(*expression.All)
		if !ok {
			return false, nil
		}

		array, ok := all.Array().(*expression.Array)
		if !ok {
			return false, nil
		}

		if !pred.Bindings().SubsetOf(array.Bindings()) {
			return false, nil
		}

		if array.When() != nil &&
			!SubsetOf(pred.Satisfies(), array.When()) {
			return false, nil
		}

		mappings := expression.Expressions{array.ValueMapping()}
		return SargableFor(pred.Satisfies(), mappings) > 0, nil
	}

	return rv
}
