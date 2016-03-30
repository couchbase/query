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
	"github.com/couchbase/query/plan"
)

type sargAny struct {
	sargDefault
}

func newSargAny(pred *expression.Any) *sargAny {
	rv := &sargAny{}
	rv.sarger = func(expr2 expression.Expression) (plan.Spans, error) {
		if SubsetOf(pred, expr2) {
			return _SELF_SPANS, nil
		}

		all, ok := expr2.(*expression.All)
		if !ok {
			return nil, nil
		}

		array, ok := all.Array().(*expression.Array)
		if !ok {
			return nil, nil
		}

		if !pred.Bindings().SubsetOf(array.Bindings()) {
			return nil, nil
		}

		if array.When() != nil &&
			!SubsetOf(pred.Satisfies(), array.When()) {
			return nil, nil
		}

		return sargFor(pred.Satisfies(), array.ValueMapping(), false)
	}

	return rv
}
