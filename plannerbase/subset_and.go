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
)

func (this *subset) VisitAnd(expr *expression.And) (interface{}, error) {
	expr2 := this.expr2
	value2 := expr2.Value()
	if value2 != nil {
		return value2.Truth(), nil
	}

	expr, _ = expression.FlattenAndNoDedup(expr)

	if expr.EquivalentTo(expr2) {
		return true, nil
	}

	for _, child := range expr.Operands() {
		if SubsetOf(child, expr2) {
			return true, nil
		}
	}

	switch expr2 := expr2.(type) {
	case *expression.And:
		for _, child2 := range expr2.Operands() {
			if !SubsetOf(expr, child2) {
				return false, nil
			}
		}

		return true, nil
	case *expression.Or:
		for _, child2 := range expr2.Operands() {
			if SubsetOf(expr, child2) {
				return true, nil
			}
		}
	}

	return false, nil
}
