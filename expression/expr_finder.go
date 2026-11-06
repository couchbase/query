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

// a reference to a LET variable whose binding contains a subquery (not inlined)
// is treated the same as a subquery
func (this *exprFinder) VisitIdentifier(expr *Identifier) (any, error) {
	if expr.IsLetSubqVariable() {
		this.hasSubq = true
	}
	return expr, nil
}

func (this *exprFinder) VisitFunction(expr Function) (any, error) {
	if _, ok := expr.(*UserDefinedFunction); ok {
		this.hasUdf = true
	}
	return expr, this.TraverseList(expr.Children())
}

// Mark references to LET variables that are not inlined (see Bindings.MappingsNoSubq()).
func MarkLetSubqVariables(expr Expression, vars map[string]bool) error {
	if expr == nil || len(vars) == 0 {
		return nil
	}
	marker := &letSubqVarMarker{vars: vars}
	marker.traverser = marker
	_, err := expr.Accept(marker)
	return err
}

type letSubqVarMarker struct {
	TraverserBase

	vars map[string]bool
}

func (this *letSubqVarMarker) VisitIdentifier(expr *Identifier) (any, error) {
	if this.vars[expr.Identifier()] {
		expr.identFlags |= IDENT_IS_LET_SUBQ_VAR
	}
	return expr, nil
}

// Does not go into subqueries: subquery expressions are not copied (the original would be
// modified), a WHERE term containing a subquery already sets hasSubq, and identifiers inside
// a subquery are resolved by the subquery's own LET processing.
func (this *letSubqVarMarker) VisitSubquery(expr Subquery) (any, error) {
	return expr, nil
}
