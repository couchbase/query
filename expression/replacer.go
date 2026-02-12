//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

/*
Replacer is used to replace one expr with another
*/

func ReplaceExpr(origExpr, oldExpr, newExpr Expression) (Expression, bool, error) {
	replacer := newReplacer(oldExpr, newExpr)
	replaceExpr, err := replacer.Map(origExpr)
	if err != nil {
		return nil, false, err
	}

	// reset the value field if expr changed
	if replacer.replaced {
		replaceExpr.ResetValue()
	}

	return replaceExpr, replacer.replaced, nil
}

type Replacer struct {
	MapperBase

	oldExpr  Expression
	newExpr  Expression
	replaced bool
}

func newReplacer(oldExpr, newExpr Expression) *Replacer {
	rv := &Replacer{
		oldExpr: oldExpr,
		newExpr: newExpr,
	}

	rv.mapFunc = func(expr Expression) (Expression, error) {
		if expr.EquivalentTo(rv.oldExpr) {
			rv.replaced = true
			return rv.newExpr, nil
		}

		return expr, expr.MapChildren(rv)
	}

	rv.mapper = rv
	return rv
}
