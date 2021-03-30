//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package planner

import (
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

// Fold redundant AND and OR terms in the predicate
func Fold(pred expression.Expression) (expression.Expression, error) {
	f := newFold()
	rv, err := pred.Accept(f)
	if err != nil {
		return nil, err
	}

	return rv.(expression.Expression), nil
}

type fold struct {
	expression.MapperBase
}

func newFold() *fold {
	rv := &fold{}

	rv.SetMapper(rv)
	rv.SetMapFunc(
		func(expr expression.Expression) (expression.Expression, error) {
			return expr, nil
		})

	return rv
}

// Logic

func (this *fold) VisitAnd(expr *expression.And) (interface{}, error) {
	operands := expr.Operands()
	found := false

outer:
	for i, lhs := range operands[0 : len(operands)-1] {
		if lhs == nil {
			continue
		}

		for j := i + 1; j < len(operands); j++ {
			rhs := operands[j]
			if rhs == nil {
				continue
			}

			if base.SubsetOf(rhs, lhs) {
				operands[i] = nil
				found = true
				continue outer
			} else if base.SubsetOf(lhs, rhs) {
				operands[j] = nil
				found = true
			}
		}
	}

	if !found {
		return expr, nil
	}

	terms := make(expression.Expressions, 0, len(operands))
	for _, op := range operands {
		if op != nil {
			terms = append(terms, op)
		}
	}

	if len(terms) == 1 {
		return terms[0], nil
	} else {
		return expression.NewAnd(terms...), nil
	}
}

func (this *fold) VisitOr(expr *expression.Or) (interface{}, error) {
	operands := expr.Operands()
	found := false

outer:
	for i, lhs := range operands[0 : len(operands)-1] {
		if lhs == nil {
			continue
		}

		for j := i + 1; j < len(operands); j++ {
			rhs := operands[j]
			if rhs == nil {
				continue
			}

			if base.SubsetOf(lhs, rhs) {
				operands[i] = nil
				found = true
				continue outer
			} else if base.SubsetOf(rhs, lhs) {
				operands[j] = nil
				found = true
			}
		}
	}

	if !found {
		return expr, nil
	}

	terms := make(expression.Expressions, 0, len(operands))
	for _, op := range operands {
		if op != nil {
			terms = append(terms, op)
		}
	}

	if len(terms) == 1 {
		return terms[0], nil
	} else {
		return expression.NewOr(terms...), nil
	}
}
