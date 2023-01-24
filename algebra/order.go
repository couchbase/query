//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"fmt"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Order by clause. Type Order is a
struct that contains the ordering terms called sort
terms.
*/
type Order struct {
	terms SortTerms
}

/*
The function NewOrder returns a pointer to the Order
struct that has its field sort terms set to the input
argument terms.
*/
func NewOrder(terms SortTerms) *Order {
	return &Order{
		terms: terms,
	}
}

/*
Copy
*/
func (this *Order) Copy() *Order {
	return &Order{
		terms: this.terms.Copy(),
	}
}

/*
Map expressions for the terms by calling MapExpressions.
*/
func (this *Order) MapExpressions(mapper expression.Mapper) error {
	return this.terms.MapExpressions(mapper)
}

/*
Returns all contained Expressions.
*/
func (this *Order) Expressions() expression.Expressions {
	return this.terms.Expressions()
}

/*
Representation as a N1QL string.
*/
func (this *Order) String() string {
	return " ORDER BY " + this.terms.String()
}

/*
Return the ordering terms (sort terms).
*/
func (this *Order) Terms() SortTerms {
	return this.terms
}

/*
It represents multiple orderby terms.
Type SortTerms is a slice of SortTerm.
*/
type SortTerms []*SortTerm

/*
Represents the ordering term in an order by clause. Type
SortTerm is a struct containing the expression and a bool
value that decides the sort order (ASC or DESC).
*/
type SortTerm struct {
	expr       expression.Expression `json:"expr"`
	descending expression.Expression `json:"desc"`
	nullsPos   expression.Expression `json:"nulls_pos"`
}

/*
The function NewSortTerm returns a pointer to the SortTerm
struct that has its fields set to the input arguments.
*/
func NewSortTerm(expr, descending, nullsPos expression.Expression) *SortTerm {
	return &SortTerm{
		expr:       expr,
		descending: descending,
		nullsPos:   nullsPos,
	}
}

/*
Copy
*/
func (this SortTerms) Copy() SortTerms {
	sterms := make(SortTerms, len(this))
	for i, s := range this {
		if s != nil {
			sterms[i] = s.Copy()
		}
	}

	return sterms
}

/*
Copy
*/
func (this *SortTerm) Copy() *SortTerm {
	rv := &SortTerm{
		expr:       this.expr.Copy(),
		descending: nil,
		nullsPos:   nil,
	}
	if this.descending != nil {
		rv.descending = this.descending.Copy()
	}
	if this.nullsPos != nil {
		rv.nullsPos = this.nullsPos.Copy()
	}
	return rv
}

/*
Representation as a N1QL string.
*/
func (this *SortTerm) String() string {
	s := expression.NewStringer().Visit(this.expr)

	d := false
	if this.Descending(nil) {
		s += " DESC"
		d = true
	}
	if this.NullsLast(nil) {
		if !d {
			s += " NULLS LAST"
		}
	} else if d {
		s += " NULLS FIRST"
	}

	return s
}

/*
Return the expression that is sorted in the order
by clause.
*/
func (this *SortTerm) Expression() expression.Expression {
	return this.expr
}

/*
Return bool value representing ASC or DESC sort order.
*/
func (this *SortTerm) Descending(context expression.Context) bool {
	if this.descending == nil {
		// optional expression missing so return default order
		return false
	}
	r, err := this.descending.Evaluate(nil, context)
	if err == nil {
		if r.Type() != value.STRING {
			if context != nil {
				ectx, ok := context.(interface{ Warning(errors.Error) })
				if ok {
					ectx.Warning(errors.NewEvaluationError(nil, fmt.Sprintf("sort order: Invalid value %v", r)))
				}
			}
			return false
		} else if s, ok := r.Actual().(string); ok {
			if strings.ToLower(s) == "desc" {
				return true
			} else if strings.ToLower(s) == "asc" {
				return false
			} else if context != nil {
				ectx, ok := context.(interface{ Warning(errors.Error) })
				if ok {
					ectx.Warning(errors.NewEvaluationError(nil, fmt.Sprintf("sort order: Invalid value %s", s)))
				}
			}
		}
	} else {
		if context != nil {
			ectx, ok := context.(interface{ Warning(errors.Error) })
			if ok {
				ectx.Warning(errors.NewEvaluationError(err, "sort order"))
			}
		}
	}
	return false
}

func (this *SortTerm) DescendingExpr() expression.Expression {
	return this.descending
}

func (this *SortTerm) NullsLast(context expression.Context) bool {
	if this.nullsPos == nil {
		// optional expression missing so return default nulls position based on order
		return this.Descending(context)
	}
	r, err := this.nullsPos.Evaluate(nil, context)
	if err == nil {
		if r.Type() != value.STRING {
			if context != nil {
				ectx, ok := context.(interface{ Warning(errors.Error) })
				if ok {
					ectx.Warning(errors.NewEvaluationError(nil,
						fmt.Sprintf("nulls sorted position: Invalid value %v", r)))
				}
			}
			return this.Descending(context)
		} else if s, ok := r.Actual().(string); ok {
			if strings.ToLower(s) == "last" {
				return true
			} else if strings.ToLower(s) == "first" {
				return false
			} else if context != nil {
				ectx, ok := context.(interface{ Warning(errors.Error) })
				if ok {
					ectx.Warning(errors.NewEvaluationError(nil,
						fmt.Sprintf("nulls sorted position: Invalid value %s", s)))
				}
			}
		}
	} else {
		if context != nil {
			ectx, ok := context.(interface{ Warning(errors.Error) })
			if ok {
				ectx.Warning(errors.NewEvaluationError(err, "nulls sorted position"))
			}
		}
	}
	// if we failed to evaluate, use the default nulls position based on order
	return this.Descending(context)
}

func (this *SortTerm) NullsPosExpr() expression.Expression {
	return this.nullsPos
}

/*
Map Expressions for all sort terms in the receiver.
*/
func (this SortTerms) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this {
		term.expr, err = mapper.Map(term.expr)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this SortTerms) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, len(this))

	for i, term := range this {
		exprs[i] = term.expr
	}

	return exprs
}

/*
Representation as a N1QL string.
*/
func (this SortTerms) String() string {
	s := ""

	for i, term := range this {
		if i > 0 {
			s += ", "
		}

		s += term.String()
	}

	return s
}
