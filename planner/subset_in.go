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

type subsetIn struct {
	subsetDefault
	in *expression.In
}

func newSubsetIn(in *expression.In) *subsetIn {
	rv := &subsetIn{
		subsetDefault: *newSubsetDefault(in),
		in:            in,
	}

	return rv
}

func (this *subsetIn) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	return expr.Operand().DependsOn(this.in.First()), nil
}

func (this *subsetIn) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	return expr.Operand().DependsOn(this.in.First()), nil
}

func (this *subsetIn) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return expr.Operand().DependsOn(this.in.First()), nil
}

func (this *subsetIn) VisitWithin(expr *expression.Within) (interface{}, error) {
	return this.in.First().EquivalentTo(expr.First()) &&
		this.in.Second().EquivalentTo(expr.Second()), nil
}
