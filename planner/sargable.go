//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
)

func SargableFor(pred, vpred expression.Expression, index datastore.Index, keys datastore.IndexKeys,
	includes expression.Expressions, missing, gsi bool, isArrays []bool, context *PrepareContext,
	aliases map[string]bool) (min, max, sum, include int, skeys []bool) {

	skeys = make([]bool, len(keys)+len(includes))

	if pred == nil && vpred == nil {
		return
	}

	if or, ok := pred.(*expression.Or); ok {
		return sargableForOr(or, vpred, index, keys, includes, missing, gsi, isArrays, context, aliases)
	}

	skiped := false

	for i := 0; i < len(keys); i++ {
		// Terminate on statically-valued expression
		if keys[i].Expr.Value() != nil {
			return
		}

		vectorType := keys[i].VectorType()

		spred := pred
		if vectorType != "" {
			spred = vpred
		}
		if spred == nil {
			if !gsi || (i == 0 && !missing) {
				return
			}
			missing = true
			skiped = true
			continue
		}

		s := &sargable{keys[i].Expr, missing, (i < len(isArrays) && isArrays[i]), gsi, vectorType,
			index, context, aliases}

		r, err := spred.Accept(s)

		if err != nil {
			return
		}

		if r.(bool) {
			max = i + 1
			skeys[i] = true
			sum++
		} else {
			if !gsi || (i == 0 && !missing) {
				return
			}
			skiped = true
		}

		if !skiped {
			min = max
		}

		if gsi {
			missing = true
		}
	}

	if pred != nil {
		for i := range includes {
			s := &sargable{includes[i], true, false, gsi, "", index, context, aliases}
			r, err := pred.Accept(s)
			if err != nil {
				return
			}
			if r.(bool) {
				include++
				skeys[i+len(keys)] = true
			}
		}
	}

	return
}

func sargableForOr(or *expression.Or, vpred expression.Expression, index datastore.Index, keys datastore.IndexKeys,
	includes expression.Expressions, missing, gsi bool, isArrays []bool, context *PrepareContext,
	aliases map[string]bool) (min, max, sum, include int, skeys []bool) {

	skeys = make([]bool, len(keys)+len(includes))

	// OR should have already been flattened with DNF transformation
	for _, c := range or.Operands() {
		cmin, cmax, csum, cinclude, cskeys := SargableFor(c, vpred, index, keys, includes,
			missing, gsi, isArrays, context, aliases)
		if (cmin == 0 && !missing) || cmax == 0 || csum < cmin {
			return 0, 0, 0, 0, skeys
		}

		if min == 0 || min > cmin {
			min = cmin
		}

		if max == 0 || max < cmax {
			max = cmax
		}

		if include == 0 || include < cinclude {
			include = cinclude
		}

		sum += csum

		for i := 0; i < len(cskeys); i++ {
			skeys[i] = skeys[i] || cskeys[i]
		}
	}

	return
}

type sargable struct {
	key        expression.Expression
	missing    bool
	array      bool
	gsi        bool
	vectorType string
	index      datastore.Index
	context    *PrepareContext
	aliases    map[string]bool
}

// Arithmetic

func (this *sargable) VisitAdd(pred *expression.Add) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitDiv(pred *expression.Div) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitMod(pred *expression.Mod) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitMult(pred *expression.Mult) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitNeg(pred *expression.Neg) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitSub(pred *expression.Sub) (any, error) {
	return this.visitDefault(pred)
}

// Case

func (this *sargable) VisitSearchedCase(pred *expression.SearchedCase) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitSimpleCase(pred *expression.SimpleCase) (any, error) {
	return this.visitDefault(pred)
}

// Collection

func (this *sargable) VisitArray(pred *expression.Array) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitEvery(pred *expression.Every) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitExists(pred *expression.Exists) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitFirst(pred *expression.First) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitObject(pred *expression.Object) (any, error) {
	return this.visitDefault(pred)
}

// Comparison

func (this *sargable) VisitBetween(pred *expression.Between) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitEq(pred *expression.Eq) (any, error) {
	return this.visitBinary(pred)
}

func (this *sargable) VisitLE(pred *expression.LE) (any, error) {
	return this.visitBinary(pred)
}

func (this *sargable) VisitLike(pred *expression.Like) (any, error) {
	return this.visitLike(pred)
}

func (this *sargable) VisitLT(pred *expression.LT) (any, error) {
	return this.visitBinary(pred)
}

func (this *sargable) VisitIsMissing(pred *expression.IsMissing) (any, error) {
	if this.missing && !this.array && pred.Operand().EquivalentTo(this.key) {
		return true, nil
	}

	return this.visitDefault(pred)
}

func (this *sargable) VisitIsNotMissing(pred *expression.IsNotMissing) (any, error) {
	return this.visitUnary(pred)
}

func (this *sargable) VisitIsNotNull(pred *expression.IsNotNull) (any, error) {
	return this.visitUnary(pred)
}

func (this *sargable) VisitIsNotValued(pred *expression.IsNotValued) (any, error) {
	if this.missing && !this.array && pred.Operand().EquivalentTo(this.key) {
		return true, nil
	}
	return this.visitDefault(pred)
}

func (this *sargable) VisitIsNull(pred *expression.IsNull) (any, error) {
	return this.visitUnary(pred)
}

func (this *sargable) VisitIsValued(pred *expression.IsValued) (any, error) {
	return this.visitUnary(pred)
}

// Concat
func (this *sargable) VisitConcat(pred *expression.Concat) (any, error) {
	return this.visitDefault(pred)
}

// Constant
func (this *sargable) VisitConstant(pred *expression.Constant) (any, error) {
	return this.visitDefault(pred)
}

// Identifier
func (this *sargable) VisitIdentifier(pred *expression.Identifier) (any, error) {
	return this.visitDefault(pred)
}

// Construction

func (this *sargable) VisitArrayConstruct(pred *expression.ArrayConstruct) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitObjectConstruct(pred *expression.ObjectConstruct) (any, error) {
	return this.visitDefault(pred)
}

// Logic

func (this *sargable) VisitNot(pred *expression.Not) (any, error) {
	return this.visitUnary(pred)
}

// Navigation

func (this *sargable) VisitElement(pred *expression.Element) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitField(pred *expression.Field) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitFieldName(pred *expression.FieldName) (any, error) {
	return this.visitDefault(pred)
}

func (this *sargable) VisitSlice(pred *expression.Slice) (any, error) {
	return this.visitDefault(pred)
}

// Self
func (this *sargable) VisitSelf(pred *expression.Self) (any, error) {
	return this.visitDefault(pred)
}

// Function
func (this *sargable) VisitFunction(pred expression.Function) (any, error) {
	switch pred := pred.(type) {
	case *expression.RegexpLike:
		return this.visitLike(pred)
	case *expression.ApproxVectorDistance:
		if this.vectorType == datastore.IK_DENSE_VECTOR_NAME {
			if index6, ok := this.index.(datastore.Index6); ok {
				fld := pred.Field()
				if fld.EquivalentTo(this.key) &&
					datastore.CompatibleMetric(index6.VectorDistanceType(), pred.Metric()) {
					return true, nil
				}
			}
		}
		return false, nil
	case *expression.SparseVectorDistance:
		if this.vectorType == datastore.IK_SPARSE_VECTOR_NAME {
			if _, ok := this.index.(datastore.Index6); ok && pred.Field().EquivalentTo(this.key) {
				return true, nil
			}
		}
		return false, nil
	}

	return this.visitDefault(pred)
}

// Subquery
func (this *sargable) VisitSubquery(pred expression.Subquery) (any, error) {
	return this.visitDefault(pred)
}

// InferUnderParenthesis
func (this *sargable) VisitParenInfer(pred expression.ParenInfer) (any, error) {
	return this.visitDefault(pred)
}

// NamedParameter
func (this *sargable) VisitNamedParameter(pred expression.NamedParameter) (any, error) {
	return this.visitDefault(pred)
}

// PositionalParameter
func (this *sargable) VisitPositionalParameter(pred expression.PositionalParameter) (any, error) {
	return this.visitDefault(pred)
}

// Cover
func (this *sargable) VisitCover(pred *expression.Cover) (any, error) {
	return pred.Covered().Accept(this)
}

// All
func (this *sargable) VisitAll(pred *expression.All) (any, error) {
	return pred.Array().Accept(this)
}

func (this *sargable) visitDefault(pred expression.Expression) (bool, error) {
	return this.defaultSargable(pred), nil
}

func (this *sargable) defaultSargable(pred expression.Expression) bool {
	return this.vectorType == "" && (base.SubsetOf(pred, this.key) ||
		((pred.PropagatesMissing() || pred.PropagatesNull()) && pred.DependsOn(this.key)))
}
