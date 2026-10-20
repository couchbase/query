//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

// Look for Subquery and UDF references
func FindExprParts(expr Expression) (bool, bool, error) {
	finder := newExprFinder()
	_, err := expr.Accept(finder)
	if err != nil {
		return false, false, err
	}
	return finder.hasSubq, finder.hasUdf, nil
}

type exprFinder struct {
	TraverserBase

	hasSubq bool
	hasUdf  bool
}

func newExprFinder() *exprFinder {
	rv := &exprFinder{}

	rv.traverser = rv
	return rv
}

func (this *exprFinder) VisitSubquery(expr Subquery) (any, error) {
	this.hasSubq = true
	return expr, nil
}

func (this *exprFinder) VisitFunction(expr Function) (any, error) {
	if _, ok := expr.(*UserDefinedFunction); ok {
		this.hasUdf = true
	}
	return expr, this.TraverseList(expr.Children())
}
