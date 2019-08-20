//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

func chkArrayKeyCover(pred Expression, keyspace string, exprs Expressions, all *All, options CoveredOptions) Covered {
	// make a copy of exprs but excludes array keys (*All expression)
	allExprs := make(Expressions, 0, len(exprs))
	for _, exp := range exprs {
		if _, ok := exp.(*All); !ok {
			allExprs = append(allExprs, exp)
		}
	}

	if array, ok := all.array.(*Array); ok {
		if array.when != nil {
			return CoveredFalse
		}

		if options.hasCoverBindExpr() {
			for _, b := range array.bindings {
				if pred.EquivalentTo(b.expr) {
					return CoveredEquiv
				}
			}
		} else if options.hasCoverSatisfies() {
			allExprs = append(allExprs, array.valueMapping)
			switch pred.CoveredBy(keyspace, allExprs, options) {
			case CoveredEquiv:
				return CoveredEquiv
			case CoveredTrue:
				return CoveredTrue
			}
		}
	} else {
		if options.hasCoverBindExpr() {
			if pred.EquivalentTo(all.array) {
				return CoveredEquiv
			}
		} else if options.hasCoverSatisfies() {
			switch pred.CoveredBy(keyspace, allExprs, options) {
			case CoveredEquiv:
				return CoveredEquiv
			case CoveredTrue:
				return CoveredTrue
			}
		}
	}

	return CoveredFalse
}

/*
Wrapper for Expression.CoveredBy - to be used by the planner
Function rather than method to make sure we don't pick up
ExpressionBase.CoveredBy() in error
*/
func IsCovered(expr Expression, keyspace string, exprs Expressions) bool {
	isCovered := expr.CoveredBy(keyspace, exprs, CoveredOptions{0})
	return isCovered == CoveredSkip || isCovered == CoveredEquiv || isCovered == CoveredTrue
}

func IsArrayCovered(expr Expression, keyspace string, exprs Expressions) bool {
	isCovered := expr.CoveredBy(keyspace, exprs, CoveredOptions{COVER_BIND_VAR | COVER_SATISFIES})
	return isCovered == CoveredSkip || isCovered == CoveredEquiv || isCovered == CoveredTrue
}
