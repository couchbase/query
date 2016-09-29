//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbase/query/value"
)

type Coverer struct {
	MapperBase

	covers       []*Cover
	filterCovers map[*Cover]value.Value
}

func NewCoverer(covers []*Cover, filterCovers map[*Cover]value.Value) *Coverer {
	rv := &Coverer{
		covers:       covers,
		filterCovers: filterCovers,
	}

	rv.mapFunc = func(expr Expression) (Expression, error) {
		for _, c := range covers {
			if c.Covered().EquivalentTo(expr) {
				return c, nil
			}
		}

		for c, _ := range filterCovers {
			if c.Covered().EquivalentTo(expr) {
				return c, nil
			}
		}

		return expr, expr.MapChildren(rv)
	}

	rv.mapper = rv
	return rv
}

func (this *Coverer) Covers() []*Cover {
	return this.covers
}

func (this *Coverer) FilterCovers() map[*Cover]value.Value {
	return this.filterCovers
}

// Constant

func (this *Coverer) VisitConstant(expr *Constant) (interface{}, error) {
	return expr, nil
}

// Parameters

func (this *Coverer) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return expr, nil
}

func (this *Coverer) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return expr, nil
}

// Cover
func (this *Coverer) VisitCover(expr *Cover) (interface{}, error) {
	return expr, nil
}
