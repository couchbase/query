//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

type Covered int

const (
	CoveredFalse    = Covered(iota) // not covered
	CoveredContinue                 // covering state can't be established yet, currently unused
	CoveredSkip                     // expression not relevant for covering, skip to next
	CoveredEquiv                    // expression is covered, ignore the rest
	CoveredTrue                     // covered
)

type CoveredOptions struct {
	coverFlags uint32
}

const (
	COVER_SKIP = 1 << iota
	COVER_TRICKLE
	COVER_BIND_VAR
	COVER_BIND_EXPR
	COVER_SATISFIES
	COVER_IMPLICIT_ARRAYKEY
	COVER_IN_SUBQUERY // Indicates if covering check is traversing down a subquery
)

const COVER_ARRAY_KEY_OPTIONS = (COVER_BIND_VAR | COVER_BIND_EXPR | COVER_SATISFIES)

func (this *CoveredOptions) hasCoverSkip() bool {
	return (this.coverFlags & COVER_SKIP) != 0
}

func (this *CoveredOptions) setCoverSkip() {
	this.coverFlags |= COVER_SKIP
}

func (this *CoveredOptions) hasCoverTrickle() bool {
	return (this.coverFlags & COVER_TRICKLE) != 0
}

func (this *CoveredOptions) setCoverTrickle() {
	this.coverFlags |= COVER_TRICKLE
}

func (this *CoveredOptions) hasCoverBindVar() bool {
	return (this.coverFlags & COVER_BIND_VAR) != 0
}

func (this *CoveredOptions) setCoverBindVar() {
	this.coverFlags |= COVER_BIND_VAR
}

func (this *CoveredOptions) unsetCoverBindVar() {
	this.coverFlags &^= COVER_BIND_VAR
}

func (this *CoveredOptions) hasCoverBindExpr() bool {
	return (this.coverFlags & COVER_BIND_EXPR) != 0
}

func (this *CoveredOptions) setCoverBindExpr() {
	this.coverFlags |= COVER_BIND_EXPR
}

func (this *CoveredOptions) unsetCoverBindExpr() {
	this.coverFlags &^= COVER_BIND_EXPR
}

func (this *CoveredOptions) hasCoverSatisfies() bool {
	return (this.coverFlags & COVER_SATISFIES) != 0
}

func (this *CoveredOptions) setCoverSatisfies() {
	this.coverFlags |= COVER_SATISFIES
}

func (this *CoveredOptions) unsetCoverSatisfies() {
	this.coverFlags &^= COVER_SATISFIES
}

func (this *CoveredOptions) hasCoverArrayKeyOptions() bool {
	return (this.coverFlags & COVER_ARRAY_KEY_OPTIONS) != 0
}

func (this *CoveredOptions) setCoverArrayKeyOptions() {
	this.coverFlags |= COVER_ARRAY_KEY_OPTIONS
}

func (this *CoveredOptions) hasCoverImplicitArrayKey() bool {
	return (this.coverFlags & COVER_IMPLICIT_ARRAYKEY) != 0
}

func (this *CoveredOptions) setCoverImplicitArrayKey() {
	this.coverFlags |= COVER_IMPLICIT_ARRAYKEY
}

func (this *CoveredOptions) unsetCoverImplicitArrayKey() {
	this.coverFlags &^= COVER_IMPLICIT_ARRAYKEY
}

func (this *CoveredOptions) SetInSubqueryFlag() {
	this.coverFlags |= COVER_IN_SUBQUERY
}

func (this *CoveredOptions) UnsetInSubqueryFlag() {
	this.coverFlags &^= COVER_IN_SUBQUERY
}

func (this *CoveredOptions) InSubqueryTraversal() bool {
	return this.coverFlags&COVER_IN_SUBQUERY != 0
}

func chkArrayKeyCover(pred Expression, keyspace string, exprs Expressions, all *All,
	options CoveredOptions) Covered {

	if options.hasCoverBindExpr() {
		if array, ok := all.array.(*Array); ok {
			for _, b := range array.bindings {
				if pred.EquivalentTo(b.expr) {
					return CoveredEquiv
				}
			}
		} else if pred.EquivalentTo(all.array) {
			return CoveredEquiv
		}
	} else if options.hasCoverSatisfies() {
		switch pred.(type) {
		case *Any, *AnyEvery, *Every:
			noptions := CoveredOptions{0}
			if options.hasCoverImplicitArrayKey() {
				noptions.setCoverImplicitArrayKey()
			}
			switch pred.CoveredBy(keyspace, exprs, noptions) {
			case CoveredEquiv:
				return CoveredEquiv
			case CoveredTrue:
				if _, ok := pred.(*Any); ok {
					return CoveredTrue
				}
			}
		}
	}

	return CoveredFalse
}

func renameBindings(other, expr Expression, copy bool) (bool, bool, Expression) {
	ie := HasRenameableBindings(expr, other, nil)
	ei := HasRenameableBindings(other, expr, nil)
	if ie == BINDING_VARS_CONFLICT || ei == BINDING_VARS_CONFLICT {
		return true, false, other
	} else if ei == BINDING_VARS_DIFFER {
		renamer := NewRenamer(getExprBindings(other), getExprBindings(expr))
		if copy {
			other = other.Copy()
		}
		rv, err := renamer.Map(other)
		return err != nil, true, rv
	}
	return false, false, other
}

/*
Wrapper for Expression.CoveredBy - to be used by the planner
Function rather than method to make sure we don't pick up
ExpressionBase.CoveredBy() in error
*/

func IsCovered(expr Expression, keyspace string, exprs Expressions,
	implicitAny bool) bool {
	options := CoveredOptions{0}
	if implicitAny {
		options.setCoverImplicitArrayKey()
	}
	isCovered := expr.CoveredBy(keyspace, exprs, options)
	return isCovered == CoveredSkip || isCovered == CoveredEquiv || isCovered == CoveredTrue
}

func IsArrayCovered(expr Expression, keyspace string, exprs Expressions) bool {
	isCovered := expr.CoveredBy(keyspace, exprs, CoveredOptions{COVER_BIND_VAR | COVER_SATISFIES})
	return isCovered == CoveredSkip || isCovered == CoveredEquiv || isCovered == CoveredTrue
}
