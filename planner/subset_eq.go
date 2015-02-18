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

type subsetEq struct {
	subsetDefault
	eq *expression.Eq
}

func newSubsetEq(eq *expression.Eq) *subsetEq {
	rv := &subsetEq{
		subsetDefault: *newSubsetDefault(eq),
		eq:            eq,
	}

	return rv
}

func (this *subsetEq) VisitLE(expr *expression.LE) (interface{}, error) {
	if this.eq.First().EquivalentTo(expr.First()) {
		return LessThanOrEquals(this.eq.Second(), expr.Second()), nil
	}

	if this.eq.Second().EquivalentTo(expr.First()) {
		return LessThanOrEquals(this.eq.First(), expr.Second()), nil
	}

	if this.eq.First().EquivalentTo(expr.Second()) {
		return LessThanOrEquals(expr.First(), this.eq.Second()), nil
	}

	if this.eq.Second().EquivalentTo(expr.Second()) {
		return LessThanOrEquals(expr.First(), this.eq.First()), nil
	}

	return false, nil
}

func (this *subsetEq) VisitLT(expr *expression.LT) (interface{}, error) {
	if this.eq.First().EquivalentTo(expr.First()) {
		return LessThan(this.eq.Second(), expr.Second()), nil
	}

	if this.eq.Second().EquivalentTo(expr.First()) {
		return LessThan(this.eq.First(), expr.Second()), nil
	}

	if this.eq.First().EquivalentTo(expr.Second()) {
		return LessThan(expr.First(), this.eq.Second()), nil
	}

	if this.eq.Second().EquivalentTo(expr.Second()) {
		return LessThan(expr.First(), this.eq.First()), nil
	}

	return false, nil
}
