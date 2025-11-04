//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func NewSimpleCoverer(covers []*Cover, filterCovers map[*Cover]value.Value) *Coverer {
	// skip covers for GROUPBY/AGGREGATES
	for i := range covers {
		if covers[i].HasExprFlag(EXPR_IS_GROUP_COVER | EXPR_IS_AGG_COVER) {
			covers = covers[:i]
			break
		}
	}
	return NewCoverer(covers, filterCovers)
}

func (this *Coverer) CoverExpr(expr Expression) (Expression, error) {
	if expr != nil {
		return this.Map(expr)
	}
	return expr, nil

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
