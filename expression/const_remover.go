//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

func RemoveConstants(expr Expression) (Expression, error) {
	switch expr := expr.(type) {
	case *And, *Or:
		// continue below
	case nil:
		return expr, nil
	default:
		val := expr.Value()
		if val != nil {
			if val.Truth() {
				return TRUE_EXPR, nil
			} else {
				return FALSE_EXPR, nil
			}
		}
		return expr, nil
	}

	consRemover := newConstRemover()
	return consRemover.Map(expr.Copy())
}

type constRemover struct {
	MapperBase
}

func newConstRemover() *constRemover {
	rv := &constRemover{}

	rv.SetMapper(rv)
	return rv
}

func (this *constRemover) VisitAnd(expr *And) (interface{}, error) {
	operands := expr.Operands()
	terms := make(Expressions, 0, len(operands))
	for _, op := range operands {
		term, err := this.Map(op)
		if err != nil {
			return MISSING_EXPR, err
		}
		val := term.Value()
		if val != nil {
			if !val.Truth() {
				return FALSE_EXPR, nil
			}
		} else {
			terms = append(terms, term)
		}
	}
	if len(terms) == 0 {
		return TRUE_EXPR, nil
	} else if len(terms) == 1 {
		return terms[0], nil
	}
	return NewAnd(terms...), nil
}

func (this *constRemover) VisitOr(expr *Or) (interface{}, error) {
	operands := expr.Operands()
	terms := make(Expressions, 0, len(operands))
	for _, op := range operands {
		term, err := this.Map(op)
		if err != nil {
			return MISSING_EXPR, err
		}
		val := term.Value()
		if val != nil {
			if val.Truth() {
				return TRUE_EXPR, nil
			}
		} else {
			terms = append(terms, term)
		}
	}
	if len(terms) == 0 {
		return FALSE_EXPR, nil
	} else if len(terms) == 1 {
		return terms[0], nil
	}
	return NewOr(terms...), nil
}
