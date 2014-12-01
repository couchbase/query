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
	"github.com/couchbaselabs/query/expression"
)

type subsetLE struct {
	subsetDefault

	le *expression.LE
}

func newSubsetLE(le *expression.LE) *subsetLE {
	rv := &subsetLE{
		subsetDefault: *newSubsetDefault(le),
		le:            le,
	}

	return rv
}

func (this *subsetLE) VisitLE(expr *expression.LE) (interface{}, error) {
	if this.le.First().EquivalentTo(expr.First()) {
		return LessThanOrEquals(this.le.Second(), expr.Second()), nil
	}

	if this.le.Second().EquivalentTo(expr.Second()) {
		return LessThanOrEquals(expr.First(), this.le.First()), nil
	}

	return false, nil
}

func (this *subsetLE) VisitLT(expr *expression.LT) (interface{}, error) {
	if this.le.First().EquivalentTo(expr.First()) {
		return LessThan(this.le.Second(), expr.Second()), nil
	}

	if this.le.Second().EquivalentTo(expr.Second()) {
		return LessThan(expr.First(), this.le.First()), nil
	}

	return false, nil
}
