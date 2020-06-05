//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

// Remove an sub-expression (subterm) from an AND expression
func RemoveExpr(expr, removeExpr Expression) (Expression, error) {
	if expr == nil || removeExpr == nil {
		return expr, nil
	}

	remover := newExprRemover(removeExpr)
	newExpr, err := expr.Accept(remover)
	if err != nil {
		return nil, err
	}

	if newExpr == nil {
		return nil, nil
	}
	return newExpr.(Expression), nil
}

type exprRemover struct {
	removeExpr Expression
}

func newExprRemover(removeExpr Expression) *exprRemover {
	return &exprRemover{
		removeExpr: removeExpr,
	}
}

/*
Only remove expression on AND boundary
*/
func (this *exprRemover) VisitAnd(expr *And) (interface{}, error) {
	var sub interface{}
	var err error

	terms := make(Expressions, 0, len(expr.Operands()))
	useNew := false
	for _, op := range expr.Operands() {
		switch op1 := op.(type) {
		case *And:
			sub, err = this.VisitAnd(op1)
			if err == nil && (sub == nil || !op1.EquivalentTo(sub.(Expression))) {
				useNew = true
			}
		default:
			sub, err = this.visitDefault(op1)
		}
		if err != nil {
			return nil, err
		}
		if sub != nil {
			terms = append(terms, sub.(Expression))
		} else {
			useNew = true
		}
	}

	if !useNew {
		// nothing changed
		return expr, nil
	}

	if len(terms) == 0 {
		return nil, nil
	} else if len(terms) == 1 {
		return terms[0], nil
	}

	return NewAnd(terms...), nil
}

// Arithmetic

func (this *exprRemover) VisitAdd(pred *Add) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitDiv(pred *Div) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitMod(pred *Mod) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitMult(pred *Mult) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitNeg(pred *Neg) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitSub(pred *Sub) (interface{}, error) {
	return this.visitDefault(pred)
}

// Case

func (this *exprRemover) VisitSearchedCase(pred *SearchedCase) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitSimpleCase(pred *SimpleCase) (interface{}, error) {
	return this.visitDefault(pred)
}

// Collection

func (this *exprRemover) VisitAny(pred *Any) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitEvery(pred *Every) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitAnyEvery(pred *AnyEvery) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitArray(pred *Array) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitFirst(pred *First) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitObject(pred *Object) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitExists(pred *Exists) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIn(pred *In) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitWithin(pred *Within) (interface{}, error) {
	return this.visitDefault(pred)
}

// Comparison

func (this *exprRemover) VisitBetween(pred *Between) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitEq(pred *Eq) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitLE(pred *LE) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitLike(pred *Like) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitLT(pred *LT) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsMissing(pred *IsMissing) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsNotMissing(pred *IsNotMissing) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsNotNull(pred *IsNotNull) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsNotValued(pred *IsNotValued) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsNull(pred *IsNull) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitIsValued(pred *IsValued) (interface{}, error) {
	return this.visitDefault(pred)
}

// Concat
func (this *exprRemover) VisitConcat(pred *Concat) (interface{}, error) {
	return this.visitDefault(pred)
}

// Constant
func (this *exprRemover) VisitConstant(pred *Constant) (interface{}, error) {
	return this.visitDefault(pred)
}

// Identifier
func (this *exprRemover) VisitIdentifier(pred *Identifier) (interface{}, error) {
	return this.visitDefault(pred)
}

// Construction

func (this *exprRemover) VisitArrayConstruct(pred *ArrayConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitObjectConstruct(pred *ObjectConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

// Logic

func (this *exprRemover) VisitOr(pred *Or) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitNot(pred *Not) (interface{}, error) {
	return this.visitDefault(pred)
}

// Navigation

func (this *exprRemover) VisitElement(pred *Element) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitField(pred *Field) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitFieldName(pred *FieldName) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) VisitSlice(pred *Slice) (interface{}, error) {
	return this.visitDefault(pred)
}

// Self
func (this *exprRemover) VisitSelf(pred *Self) (interface{}, error) {
	return this.visitDefault(pred)
}

// Function
func (this *exprRemover) VisitFunction(pred Function) (interface{}, error) {
	return this.visitDefault(pred)
}

// Subquery
func (this *exprRemover) VisitSubquery(pred Subquery) (interface{}, error) {
	return this.visitDefault(pred)
}

// NamedParameter
func (this *exprRemover) VisitNamedParameter(pred NamedParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// PositionalParameter
func (this *exprRemover) VisitPositionalParameter(pred PositionalParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// Cover
func (this *exprRemover) VisitCover(pred *Cover) (interface{}, error) {
	return this.visitDefault(pred)
}

// All
func (this *exprRemover) VisitAll(pred *All) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprRemover) visitDefault(expr Expression) (interface{}, error) {
	if expr.EquivalentTo(this.removeExpr) {
		return nil, nil
	}
	return expr, nil
}
