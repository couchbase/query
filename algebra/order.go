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

func (this *Order) HasVectorTerm() bool {
	for _, term := range this.terms {
		if term.IsVectorTerm() {
			return true
		}
	}
	return false
}

func (this *Order) HasProjectionAlias() bool {
	for _, term := range this.terms {
		if term.IsProjectionAlias() {
			return true
		}
	}
	return false
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
	rv := &SortTerm{
		expr:       expr,
		descending: descending,
		nullsPos:   nullsPos,
	}
	expr.SetExprFlag(expression.EXPR_ORDER_BY)
	switch expr.(type) {
	case *expression.ApproxVectorDistance, *expression.VectorDistance:
		// Add NULLS LAST for ASC collation of Distance functions
		if nullsPos == nil {
			var collation string
			if descending == nil {
				collation = "asc"
			} else {
				dv := descending.Value()
				if dv != nil && dv.Type() == value.STRING {
					collation = strings.ToLower(dv.ToString())
				}
			}
			if collation == "asc" {
				rv.nullsPos = expression.NewConstant("LAST")
			}
		}
	}
	return rv
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
	stringer := expression.NewStringer()
	stringer.VisitShared(this.expr)

	d := false
	dDynamic := false

	if dExpr := this.descending; dExpr != nil {
		// since we only accept constant, named/positional parameters, function parameters as ORDER BY direction
		// if the Value() is not nil, then it is a Constant and the direction can be evaluated at this stage.
		if dExpr.Value() != nil {
			if this.Descending(nil, nil) {
				stringer.WriteString(" DESC")
				d = true
			}
		} else {
			stringer.WriteString(" ")
			stringer.VisitShared(dExpr)
			dDynamic = true
		}
	}

	if nExpr := this.nullsPos; nExpr != nil {
		// since we only accept constant, named/positional parameters, function parameters as nulls position
		// if the Value() is not nil, then it is a Constant and the nulls position can be evaluated at this stage.
		if nExpr.Value() != nil {
			nEval := this.NullsLast(nil, nil)

			// if the ORDER BY direction is a constant, the direction is known.
			// if the Nulls position is also a constant, only write the nulls position in the string if it is not
			// the default position for the direction.
			// i.e if descending direction NULLS LAST is already implied
			// and if ascending direction NULLS FIRST is already implied
			if !dDynamic {
				if nEval {
					if !d {
						stringer.WriteString(" NULLS LAST")
					}
				} else if d {
					stringer.WriteString(" NULLS FIRST")
				}
			} else {
				if nEval {
					stringer.WriteString(" NULLS LAST")
				} else {
					stringer.WriteString(" NULLS FIRST")
				}
			}
		} else {
			stringer.WriteString(" NULLS ")
			stringer.VisitShared(nExpr)
		}

	}
	return stringer.String()
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
Returns the default sort order if expression evaluation fails.
*/
func (this *SortTerm) Descending(item value.Value, context expression.Context) bool {
	if this.descending == nil {
		// optional expression missing so return default order
		return false
	}

	r, err := this.descending.Evaluate(item, context)

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

	// return default sort order if evaluation fails
	return false
}

func (this *SortTerm) DescendingExpr() expression.Expression {
	return this.descending
}

/*
Return bool value representing NULLS LAST or NULLS FIRST nulls position.
Returns the default NULLS position based on the term's sort order if the expression evaluation fails.
*/
func (this *SortTerm) NullsLast(item value.Value, context expression.Context) bool {
	if this.nullsPos == nil {
		// optional expression missing so return default nulls position based on order
		return this.Descending(item, context)
	}

	r, err := this.nullsPos.Evaluate(item, context)

	if err == nil {
		if r.Type() != value.STRING {
			if context != nil {
				ectx, ok := context.(interface{ Warning(errors.Error) })
				if ok {
					ectx.Warning(errors.NewEvaluationError(nil,
						fmt.Sprintf("nulls sorted position: Invalid value %v", r)))
				}
			}
			return this.Descending(item, context)
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
	return this.Descending(item, context)
}

func (this *SortTerm) NullsPosExpr() expression.Expression {
	return this.nullsPos
}

func (this *SortTerm) IsVectorTerm() bool {
	_, ok := this.expr.(*expression.ApproxVectorDistance)
	return ok
}

func (this *SortTerm) IsProjectionAlias() bool {
	// only returns true when the sort term is just a projection alias (identifier),
	// it does not check whether an identifier is embedded in the sort expression
	if ident, ok := this.expr.(*expression.Identifier); ok {
		return ident.IsProjectionAlias()
	}
	return false
}

/*
Map Expressions for all sort terms in the receiver.
Maps sort term, sort direction and nulls position.
*/
func (this SortTerms) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this {
		term.expr, err = mapper.Map(term.expr)

		if err != nil {
			return
		}

		if term.descending != nil {
			term.descending, err = mapper.Map(term.descending)
			if err != nil {
				return
			}
		}

		if term.nullsPos != nil {
			term.nullsPos, err = mapper.Map(term.nullsPos)
			if err != nil {
				return
			}
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
