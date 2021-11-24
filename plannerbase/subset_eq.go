//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/search"
	"github.com/couchbase/query/value"
)

func (this *subset) VisitEq(expr *expression.Eq) (interface{}, error) {
	switch expr2 := this.expr2.(type) {

	case *expression.LE:
		if expr.First().EquivalentTo(expr2.First()) {
			return LessThanOrEquals(expr.Second(), expr2.Second()), nil
		}

		if expr.Second().EquivalentTo(expr2.First()) {
			return LessThanOrEquals(expr.First(), expr2.Second()), nil
		}

		if expr.First().EquivalentTo(expr2.Second()) {
			return LessThanOrEquals(expr2.First(), expr.Second()), nil
		}

		if expr.Second().EquivalentTo(expr2.Second()) {
			return LessThanOrEquals(expr2.First(), expr.First()), nil
		}

		return false, nil

	case *expression.LT:
		if expr.First().EquivalentTo(expr2.First()) {
			return LessThan(expr.Second(), expr2.Second()), nil
		}

		if expr.Second().EquivalentTo(expr2.First()) {
			return LessThan(expr.First(), expr2.Second()), nil
		}

		if expr.First().EquivalentTo(expr2.Second()) {
			return LessThan(expr2.First(), expr.Second()), nil
		}

		if expr.Second().EquivalentTo(expr2.Second()) {
			return LessThan(expr2.First(), expr.First()), nil
		}

		return false, nil

	case *expression.In:
		acons, ok := expr2.Second().(*expression.ArrayConstruct)
		if !ok {
			return false, nil
		}

		var rhs expression.Expression
		if expr.First().EquivalentTo(expr2.First()) {
			rhs = expr.Second()
		} else if expr.Second().EquivalentTo(expr2.First()) {
			rhs = expr.First()
		} else {
			return false, nil
		}

		for _, op := range acons.Operands() {
			if rhs.EquivalentTo(op) {
				return true, nil
			}
		}

		return false, nil

	case *expression.Within:
		acons, ok := expr2.Second().(*expression.ArrayConstruct)
		if !ok {
			return false, nil
		}

		var rhs expression.Expression
		if expr.First().EquivalentTo(expr2.First()) {
			rhs = expr.Second()
		} else if expr.Second().EquivalentTo(expr2.First()) {
			rhs = expr.First()
		} else {
			return false, nil
		}

		for _, op := range acons.Operands() {
			if rhs.EquivalentTo(op) {
				return true, nil
			}
		}

		return false, nil

	case *search.Search:
		var val value.Value

		if expr.First().EquivalentTo(expr2) {
			val = expr.Second().Value()
		} else if expr.Second().EquivalentTo(expr2) {
			val = expr.First().Value()
		}

		if val != nil && val.Type() == value.BOOLEAN && val.Truth() {
			return true, nil
		}

		return this.visitDefault(expr)

	default:
		return this.visitDefault(expr)
	}
}
