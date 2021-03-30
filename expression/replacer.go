//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package expression

/*
Replacer is used to replace one expr with another
*/

func ReplaceExpr(origExpr, oldExpr, newExpr Expression) (Expression, error) {
	replacer := newReplacer(oldExpr, newExpr)
	replaceExpr, err := replacer.Map(origExpr)
	if err != nil {
		return nil, err
	}

	// reset the value field since expr might have changed
	replaceExpr.ResetValue()

	return replaceExpr, nil
}

type Replacer struct {
	MapperBase

	oldExpr Expression
	newExpr Expression
}

func newReplacer(oldExpr, newExpr Expression) *Replacer {
	rv := &Replacer{
		oldExpr: oldExpr,
		newExpr: newExpr,
	}

	rv.mapFunc = func(expr Expression) (Expression, error) {
		if expr.EquivalentTo(rv.oldExpr) {
			return rv.newExpr, nil
		}

		return expr, expr.MapChildren(rv)
	}

	rv.mapper = rv
	return rv
}
