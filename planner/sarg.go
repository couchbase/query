//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"encoding/json"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

func SargFor(pred expression.Expression, exprs expression.Expressions) (Spans, error) {
	n := SargableFor(pred, exprs)
	s := newSarg(pred)
	s.SetMissingHigh(n < len(exprs))
	var ns Spans

	// Sarg compositive indexes right to left
	for i := n - 1; i >= 0; i-- {
		r, err := exprs[i].Accept(s)
		if err != nil || r == nil {
			return nil, err
		}

		rs := r.(Spans)
		if len(rs) == 0 {
			// Should not reach here
			return nil, nil
		}

		// Notify prev key that this key is missing a high bound
		if i > 0 {
			s.SetMissingHigh(false)
			for _, prev := range rs {
				if len(prev.Range.High) == 0 {
					s.SetMissingHigh(true)
					break
				}
			}
		}

		if ns == nil {
			// First iteration
			ns = rs
			continue
		}

		// Cross product of prev and next spans
		sp := make(Spans, 0, len(rs)*len(ns))
	prevs:
		for _, prev := range rs {
			// Full span subsumes others
			if len(prev.Range.Low) == 0 && len(prev.Range.High) == 0 {
				sp = append(sp, prev)
				continue
			}

			for _, next := range ns {
				// Full span subsumes others
				if len(next.Range.Low) == 0 && len(next.Range.High) == 0 {
					sp = append(sp, prev)
					continue prevs
				}
			}

			for j, next := range ns {
				pre := prev
				if j < len(ns)-1 {
					pre = pre.Copy()
				}

				if len(pre.Range.Low) > 0 && len(next.Range.Low) > 0 {
					pre.Range.Low = append(pre.Range.Low, next.Range.Low...)
					pre.Range.Inclusion = datastore.LOW & pre.Range.Inclusion & next.Range.Inclusion
				}

				if len(pre.Range.High) > 0 && len(next.Range.High) > 0 {
					pre.Range.High = append(pre.Range.High, next.Range.High...)
					pre.Range.Inclusion = datastore.HIGH & pre.Range.Inclusion & next.Range.Inclusion
				}

				sp = append(sp, pre)
			}
		}

		ns = sp
	}

	return ns, nil
}

func sargFor(pred, expr expression.Expression, missingHigh bool) (Spans, error) {
	s := newSarg(pred)
	s.SetMissingHigh(missingHigh)

	r, err := expr.Accept(s)
	if err != nil || r == nil {
		return nil, err
	}

	rs := r.(Spans)
	return rs, nil
}

func newSarg(pred expression.Expression) sarg {
	s, _ := pred.Accept(_SARG_FACTORY)
	return s.(sarg)
}

type sarg interface {
	expression.Visitor
	SetMissingHigh(bool)
	MissingHigh() bool
}

type sargBase struct {
	sarger      sargFunc
	missingHigh bool
}

func (this *sargBase) SetMissingHigh(v bool) {
	this.missingHigh = v
}

func (this *sargBase) MissingHigh() bool {
	return this.missingHigh
}

type sargFunc func(expression.Expression) (Spans, error)

// Arithmetic

func (this *sargBase) VisitAdd(expr *expression.Add) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitDiv(expr *expression.Div) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitMod(expr *expression.Mod) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitMult(expr *expression.Mult) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitNeg(expr *expression.Neg) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitSub(expr *expression.Sub) (interface{}, error) {
	return this.sarger(expr)
}

// Case

func (this *sargBase) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return this.sarger(expr)
}

// Collection

func (this *sargBase) VisitAny(expr *expression.Any) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitArray(expr *expression.Array) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitEvery(expr *expression.Every) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitExists(expr *expression.Exists) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitFirst(expr *expression.First) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitIn(expr *expression.In) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitWithin(expr *expression.Within) (interface{}, error) {
	return this.sarger(expr)
}

// Comparison

func (this *sargBase) VisitBetween(expr *expression.Between) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitEq(expr *expression.Eq) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitLE(expr *expression.LE) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitLike(expr *expression.Like) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitLT(expr *expression.LT) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitIsNotValued(expr *expression.IsNotValued) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return this.sarger(expr)
}

// Concat
func (this *sargBase) VisitConcat(expr *expression.Concat) (interface{}, error) {
	return this.sarger(expr)
}

// Constant
func (this *sargBase) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return this.sarger(expr)
}

// Identifier
func (this *sargBase) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	return this.sarger(expr)
}

// Construction

func (this *sargBase) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	return this.sarger(expr)
}

// Logic

func (this *sargBase) VisitAnd(expr *expression.And) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitNot(expr *expression.Not) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitOr(expr *expression.Or) (interface{}, error) {
	return this.sarger(expr)
}

// Navigation

func (this *sargBase) VisitElement(expr *expression.Element) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitField(expr *expression.Field) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	return this.sarger(expr)
}

func (this *sargBase) VisitSlice(expr *expression.Slice) (interface{}, error) {
	return this.sarger(expr)
}

// Function
func (this *sargBase) VisitFunction(expr expression.Function) (interface{}, error) {
	return this.sarger(expr)
}

// Subquery
func (this *sargBase) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	return this.sarger(expr)
}

// NamedParameter
func (this *sargBase) VisitNamedParameter(expr expression.NamedParameter) (interface{}, error) {
	return this.sarger(expr)
}

// PositionalParameter
func (this *sargBase) VisitPositionalParameter(expr expression.PositionalParameter) (interface{}, error) {
	return this.sarger(expr)
}

// Spans implements json.Unmarshaller to enable prepared statement execution
func (this Spans) UnmarshalJSON(body []byte) error {
	var _unmarshalled []*struct {
		Seek  []string
		Range struct {
			Low       []string
			High      []string
			Inclusion datastore.Inclusion
		}
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this = make(Spans, len(_unmarshalled))
	for i, span := range _unmarshalled {
		var s Span
		s.Seek = make(expression.Expressions, len(span.Seek))
		for j, seekExpr := range span.Seek {
			s.Seek[j], err = parser.Parse(seekExpr)
			if err != nil {
				return err
			}

			s.Range.Low = make(expression.Expressions, len(span.Range.Low))
			for l, lowExpr := range span.Range.Low {
				s.Range.Low[l], err = parser.Parse(lowExpr)
				if err != nil {
					return err
				}
			}

			s.Range.High = make(expression.Expressions, len(span.Range.High))
			for h, hiExpr := range span.Range.High {
				s.Range.Low[h], err = parser.Parse(hiExpr)
				if err != nil {
					return err
				}
			}

			s.Range.Inclusion = span.Range.Inclusion
		}

		this[i] = &s
	}

	return nil
}
