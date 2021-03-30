//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package plannerbase

import (
	"github.com/couchbase/query/expression"
)

func (this *subset) VisitLE(expr *expression.LE) (interface{}, error) {
	switch expr2 := this.expr2.(type) {

	case *expression.LE:
		if expr.First().EquivalentTo(expr2.First()) {
			return LessThanOrEquals(expr.Second(), expr2.Second()), nil
		}

		if expr.Second().EquivalentTo(expr2.Second()) {
			return LessThanOrEquals(expr2.First(), expr.First()), nil
		}

		return false, nil

	case *expression.LT:
		if expr.First().EquivalentTo(expr2.First()) {
			return LessThan(expr.Second(), expr2.Second()), nil
		}

		if expr.Second().EquivalentTo(expr2.Second()) {
			return LessThan(expr2.First(), expr.First()), nil
		}

		return false, nil

	default:
		return this.visitDefault(expr)
	}
}
