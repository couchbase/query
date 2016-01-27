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

type subsetEvery struct {
	subsetDefault
	every *expression.Every
}

func newSubsetEvery(every *expression.Every) *subsetEvery {
	rv := &subsetEvery{
		subsetDefault: *newSubsetDefault(every),
		every:         every,
	}

	return rv
}

func (this *subsetEvery) VisitEvery(expr *expression.Every) (interface{}, error) {
	return this.every.Bindings().EquivalentTo(expr.Bindings()) &&
		SubsetOf(this.every.Satisfies(), expr.Satisfies()), nil
}
