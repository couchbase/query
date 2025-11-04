//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
)

type PartialAggCoverer struct {
	expression.MapperBase

	covers     []*expression.Cover
	matchCover *expression.Cover
	aggs       algebra.Aggregates
}

func NewPartialAggCoverer(covers []*expression.Cover, aggs algebra.Aggregates) *PartialAggCoverer {
	rv := &PartialAggCoverer{
		covers: covers,
		aggs:   aggs,
	}

	rv.SetMapper(rv)
	rv.SetMapFunc(func(expr expression.Expression) (expression.Expression, error) {

		if rv.matchCover != nil {
			return rv.matchCover, nil
		}

		if _, ok := expr.(*expression.Cover); ok {
			return expr, nil
		}

		prevMatchCover := rv.matchCover
		defer func() {
			rv.matchCover = prevMatchCover
		}()
		rv.matchCover = nil

		if agg, ok := expr.(algebra.Aggregate); ok && !agg.IsWindowAggregate() {
			switch agg.(type) {
			case *algebra.Avg:
				return indexPartialAggregateAvg2DivisionRewrite(agg, rv.aggs)
			}

			for _, c := range covers {
				if expression.Equivalent(agg.Operands()[0], c) {
					return indexPartialAggregateCount2SumRewrite(agg, c), nil
				}

				if cagg, ok := c.Covered().(algebra.Aggregate); ok {
					if agg.EquivalentTo(cagg) && !agg.HasFlags(algebra.AGGREGATE_REWRITE_INDEX_AGGS) {
						agg1 := agg.Copy().(algebra.Aggregate)
						rv.matchCover = c
						err := agg1.MapChildren(rv)
						if err == nil {
							agg1 = indexPartialAggregateCount2SumRewrite(agg1, c)
							agg1.AddFlags(algebra.AGGREGATE_REWRITE_INDEX_AGGS)
						}
						return agg1, err
					}
				}
			}
			return expr, nil
		}

		return expr, expr.MapChildren(rv)
	})

	return rv
}

func (this *PartialAggCoverer) Covers() []*expression.Cover {
	return this.covers
}

// Parameters

func (this *PartialAggCoverer) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return expr, nil
}

func (this *PartialAggCoverer) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return expr, nil
}

type FullAggCoverer struct {
	expression.MapperBase

	covers []*expression.Cover
}

func NewFullAggCoverer(covers []*expression.Cover) *FullAggCoverer {
	rv := &FullAggCoverer{
		covers: covers,
	}

	rv.SetMapper(rv)
	rv.SetMapFunc(func(expr expression.Expression) (expression.Expression, error) {

		if _, ok := expr.(*expression.Cover); ok {
			return expr, nil
		}

		if agg, ok := expr.(algebra.Aggregate); ok && !agg.IsWindowAggregate() {
			switch agg.(type) {
			case *algebra.Avg:
				return indexFullAggregateAvg2DivisionRewrite(agg, covers)
			}

			for _, c := range covers {
				if cagg, ok := c.Covered().(algebra.Aggregate); ok {
					if agg.EquivalentTo(cagg) {
						return c, nil
					}
				}
			}
			return expr, nil
		}

		return expr, expr.MapChildren(rv)
	})

	return rv
}

func (this *FullAggCoverer) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return expr, nil
}

func (this *FullAggCoverer) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return expr, nil
}
