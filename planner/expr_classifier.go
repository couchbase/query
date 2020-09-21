//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	base "github.com/couchbase/query/plannerbase"
	"github.com/couchbase/query/value"
)

// breaks expr on AND boundaries and classify into appropriate keyspaces
func ClassifyExpr(expr expression.Expression, baseKeyspaces map[string]*base.BaseKeyspace,
	keyspaceNames map[string]string, isOnclause, doSelec, advisorValidate bool) (value.Value, error) {

	if len(baseKeyspaces) == 0 {
		return nil, errors.NewPlanError(nil, "ClassifyExpr: invalid argument baseKeyspaces")
	}

	classifier := newExprClassifier(baseKeyspaces, keyspaceNames, isOnclause, doSelec, advisorValidate)
	_, err := expr.Accept(classifier)
	if err != nil {
		return nil, err
	}

	var constant value.Value
	if classifier.constant != nil {
		if classifier.constant.Truth() {
			if !classifier.nonConstant {
				constant = classifier.constant
			}
		} else {
			constant = classifier.constant
		}
	}

	return constant, nil
}

type exprClassifier struct {
	baseKeyspaces   map[string]*base.BaseKeyspace
	keyspaceNames   map[string]string
	recursion       bool
	recurseExpr     expression.Expression
	recursionJoin   bool
	recurseJoinExpr expression.Expression
	isOnclause      bool
	nonConstant     bool
	constant        value.Value
	doSelec         bool
	advisorValidate bool
}

func newExprClassifier(baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string,
	isOnclause, doSelec, advisorValidate bool) *exprClassifier {

	return &exprClassifier{
		baseKeyspaces:   baseKeyspaces,
		keyspaceNames:   keyspaceNames,
		isOnclause:      isOnclause,
		doSelec:         doSelec,
		advisorValidate: advisorValidate,
	}
}

func (this *exprClassifier) addConstant(result bool) {
	if this.constant == nil {
		if result {
			this.constant = value.TRUE_VALUE
		} else {
			this.constant = value.FALSE_VALUE
		}
	} else {
		// true AND true is true, true AND false is false, false AND false is false
		// thus if result is true, no change to existing value
		// if result is false, change existing constant to FALSE
		if !result {
			this.constant = value.FALSE_VALUE
		}
	}
}

func (this *exprClassifier) VisitAnd(expr *expression.And) (interface{}, error) {

	var err error
	for _, op := range expr.Operands() {
		switch op := op.(type) {
		case *expression.And:
			_, err = this.VisitAnd(op)
		default:
			_, err = this.visitDefault(op)
		}
		if err != nil {
			return nil, err
		}
	}

	return expr, nil
}

func (this *exprClassifier) VisitOr(expr *expression.Or) (interface{}, error) {

	or, truth := expression.FlattenOr(expr)
	if truth {
		this.addConstant(true)
		return expression.TRUE_EXPR, nil
	}

	newExpr := false
	orTerms := make(expression.Expressions, 0, len(or.Operands()))
	for _, op := range or.Operands() {
		skip := false
		cop := op.Value()
		if op.HasExprFlag(expression.EXPR_VALUE_MISSING) || op.HasExprFlag(expression.EXPR_VALUE_NULL) {
			skip = true
		} else if cop != nil {
			// TRUE subterm should have been handled above
			if !cop.Truth() {
				// FALSE subterm can be skipped
				skip = true
			}
		}
		if skip {
			newExpr = true
		} else {
			orTerms = append(orTerms, op)
		}
	}

	if newExpr {
		if len(orTerms) == 0 {
			// expr is FALSE if all subterms skipped
			this.addConstant(false)
			return expression.FALSE_EXPR, nil
		} else {
			return this.visitDefault(expression.NewOr(orTerms...))
		}
	}

	return this.visitDefault(or)
}

// Arithmetic

func (this *exprClassifier) VisitAdd(pred *expression.Add) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitDiv(pred *expression.Div) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitMod(pred *expression.Mod) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitMult(pred *expression.Mult) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitNeg(pred *expression.Neg) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitSub(pred *expression.Sub) (interface{}, error) {
	return this.visitDefault(pred)
}

// Case

func (this *exprClassifier) VisitSearchedCase(pred *expression.SearchedCase) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitSimpleCase(pred *expression.SimpleCase) (interface{}, error) {
	return this.visitDefault(pred)
}

// Collection

func (this *exprClassifier) VisitAny(pred *expression.Any) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitEvery(pred *expression.Every) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitAnyEvery(pred *expression.AnyEvery) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitArray(pred *expression.Array) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitFirst(pred *expression.First) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitObject(pred *expression.Object) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitExists(pred *expression.Exists) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIn(pred *expression.In) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitWithin(pred *expression.Within) (interface{}, error) {
	return this.visitDefault(pred)
}

// Comparison

func (this *exprClassifier) VisitBetween(pred *expression.Between) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitEq(pred *expression.Eq) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitLE(pred *expression.LE) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitLike(pred *expression.Like) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitLT(pred *expression.LT) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsMissing(pred *expression.IsMissing) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsNotMissing(pred *expression.IsNotMissing) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsNotNull(pred *expression.IsNotNull) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsNotValued(pred *expression.IsNotValued) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsNull(pred *expression.IsNull) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitIsValued(pred *expression.IsValued) (interface{}, error) {
	return this.visitDefault(pred)
}

// Concat
func (this *exprClassifier) VisitConcat(pred *expression.Concat) (interface{}, error) {
	return this.visitDefault(pred)
}

// Constant
func (this *exprClassifier) VisitConstant(pred *expression.Constant) (interface{}, error) {
	return this.visitDefault(pred)
}

// Identifier
func (this *exprClassifier) VisitIdentifier(pred *expression.Identifier) (interface{}, error) {
	return this.visitDefault(pred)
}

// Construction

func (this *exprClassifier) VisitArrayConstruct(pred *expression.ArrayConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitObjectConstruct(pred *expression.ObjectConstruct) (interface{}, error) {
	return this.visitDefault(pred)
}

// Logic

func (this *exprClassifier) VisitNot(pred *expression.Not) (interface{}, error) {
	return this.visitDefault(pred)
}

// Navigation

func (this *exprClassifier) VisitElement(pred *expression.Element) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitField(pred *expression.Field) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitFieldName(pred *expression.FieldName) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) VisitSlice(pred *expression.Slice) (interface{}, error) {
	return this.visitDefault(pred)
}

// Self
func (this *exprClassifier) VisitSelf(pred *expression.Self) (interface{}, error) {
	return this.visitDefault(pred)
}

// Function
func (this *exprClassifier) VisitFunction(pred expression.Function) (interface{}, error) {
	return this.visitDefault(pred)
}

// Subquery
func (this *exprClassifier) VisitSubquery(pred expression.Subquery) (interface{}, error) {
	return this.visitDefault(pred)
}

// NamedParameter
func (this *exprClassifier) VisitNamedParameter(pred expression.NamedParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// PositionalParameter
func (this *exprClassifier) VisitPositionalParameter(pred expression.PositionalParameter) (interface{}, error) {
	return this.visitDefault(pred)
}

// Cover
func (this *exprClassifier) VisitCover(pred *expression.Cover) (interface{}, error) {
	return this.visitDefault(pred)
}

// All
func (this *exprClassifier) VisitAll(pred *expression.All) (interface{}, error) {
	return this.visitDefault(pred)
}

func (this *exprClassifier) visitDefault(expr expression.Expression) (interface{}, error) {

	cpred := expr.Value()
	if cpred != nil {
		if cpred.Truth() {
			this.addConstant(true)
		} else {
			this.addConstant(false)
		}
		return expr, nil
	} else {
		this.nonConstant = true
	}

	keyspaces, err := expression.CountKeySpaces(expr, this.keyspaceNames)
	if err != nil {
		return nil, err
	}

	if len(keyspaces) < 1 {
		return expr, nil
	}

	// perform expression transformation, but no DNF transformation
	dnfExpr := expr.Copy()
	dnf := base.NewDNF(dnfExpr, true, false)
	dnfExpr, err = dnf.Map(dnfExpr)
	if err != nil {
		return nil, err
	}

	// if expression transformation generates new AND terms, recurse
	if and, ok := dnfExpr.(*expression.And); ok {
		if len(keyspaces) == 1 {
			recursion := this.recursion
			defer func() { this.recursion = recursion }()
			this.recursion = true
			if this.recurseExpr == nil {
				this.recurseExpr = expr
			}
		} else {
			recursionJoin := this.recursionJoin
			defer func() { this.recursionJoin = recursionJoin }()
			this.recursionJoin = true
			if this.recurseJoinExpr == nil {
				this.recurseJoinExpr = expr
			}
		}
		return this.VisitAnd(and)
	}

	var origExpr expression.Expression
	if len(keyspaces) == 1 {
		if this.recursion {
			// recurseExpr is only used once, even through multiple recursions
			if this.recurseExpr != nil {
				origExpr = this.recurseExpr
				this.recurseExpr = nil
			}
		} else {
			origExpr = expr
		}
	} else {
		if this.recursionJoin {
			// recurseJoinExpr is only used once, even through multiple recursions
			if this.recurseJoinExpr != nil {
				origExpr = this.recurseJoinExpr
				this.recurseJoinExpr = nil
			}
		} else {
			origExpr = expr
		}
	}

	origKeyspaces := make(map[string]string, len(keyspaces))
	for a, k := range keyspaces {
		origKeyspaces[a] = k
	}

	subqueries, err := expression.ListSubqueries(expression.Expressions{expr}, false)
	if err != nil {
		return nil, err
	}

	if this.isOnclause {
		// remove references to keyspaces that's already processed
		for kspace, _ := range keyspaces {
			if baseKspace, ok := this.baseKeyspaces[kspace]; ok {
				if baseKspace.PlanDone() {
					delete(keyspaces, kspace)
				}
			}
		}
	}

	for kspace, _ := range keyspaces {
		if baseKspace, ok := this.baseKeyspaces[kspace]; ok {
			filter := base.NewFilter(dnfExpr, origExpr, keyspaces, origKeyspaces,
				this.isOnclause, len(origKeyspaces) > 1)
			if this.doSelec && !baseKspace.IsPrimaryUnnest() {
				selec, arrSelec := optExprSelec(origKeyspaces, dnfExpr, this.advisorValidate)
				filter.SetSelec(selec)
				filter.SetArraySelec(arrSelec)
				setAdjSel := true
				if anyEvery, ok := filter.FltrExpr().(*expression.AnyEvery); ok && selec > 0.0 {
					// for ANY and EVERY predicate, also calculate selectivity
					// for just ANY (used in index scan)
					any := expression.NewAny(anyEvery.Bindings(), anyEvery.Satisfies())
					selec1, arrSelec1 := optExprSelec(origKeyspaces, any, this.advisorValidate)
					if selec1 > 0.0 && arrSelec1 > 0.0 {
						filter.SetArraySelec(arrSelec1)
						filter.SetAdjSelec(selec / selec1)
						filter.SetAdjustedSelec()
						setAdjSel = false
					}
				}
				if setAdjSel {
					filter.SetAdjSelec(OPT_SELEC_NOT_AVAIL)
				}
				filter.SetSelecDone()
			}
			if len(subqueries) > 0 {
				filter.SetSubq()
			}

			if len(keyspaces) == 1 {
				baseKspace.AddFilter(filter)
			} else {
				baseKspace.AddJoinFilter(filter)
				// if this is an OR join predicate, attempt to extract a new OR-predicate
				// for a single keyspace (to enable union scan)
				if or, ok := dnfExpr.(*expression.Or); ok {
					newPred, newOrigPred, orIsJoin, err := this.extractExpr(or, baseKspace.Name())
					if err != nil {
						return nil, err
					}
					if newPred != nil {
						newKeyspaces := make(map[string]string, 1)
						newKeyspaces[baseKspace.Name()] = baseKspace.Keyspace()
						newOrigKeyspaces := make(map[string]string, 1)
						newOrigKeyspaces[baseKspace.Name()] = baseKspace.Keyspace()
						newFilter := base.NewFilter(newPred, newOrigPred, newKeyspaces,
							newOrigKeyspaces, this.isOnclause, orIsJoin)
						baseKspace.AddFilter(newFilter)
					}
				}
			}
		} else {
			return nil, errors.NewPlanInternalError(fmt.Sprintf("exprClassifier.visitDefault: missing keyspace %s", kspace))
		}
	}

	return expr, nil
}

func (this *exprClassifier) extractExpr(or *expression.Or, keyspaceName string) (
	expression.Expression, expression.Expression, bool, error) {

	orTerms, truth := expression.FlattenOr(or)
	if orTerms == nil || truth {
		return nil, nil, false, nil
	}

	var newTerm, newOrigTerm expression.Expression
	var newTerms, newOrigTerms expression.Expressions
	var isJoin = false
	for _, op := range orTerms.Operands() {
		baseKeyspaces := base.CopyBaseKeyspaces(this.baseKeyspaces)
		_, err := ClassifyExpr(op, baseKeyspaces, this.keyspaceNames, this.isOnclause, this.doSelec, this.advisorValidate)
		if err != nil {
			return nil, nil, false, err
		}
		newTerm = nil
		newOrigTerm = nil
		if kspace, ok := baseKeyspaces[keyspaceName]; ok {
			for _, fl := range kspace.Filters() {
				fltrExpr := fl.FltrExpr()
				origExpr := fl.OrigExpr()

				if newTerm == nil {
					newTerm = fltrExpr
				} else {
					newTerm = expression.NewAnd(newTerm, fltrExpr)
				}

				if origExpr != nil {
					if newOrigTerm == nil {
						newOrigTerm = origExpr
					} else {
						newOrigTerm = expression.NewAnd(newOrigTerm, origExpr)
					}
				}

				isJoin = isJoin || fl.IsJoin()
			}
		}

		if newTerm != nil {
			newTerms = append(newTerms, newTerm)
			if newOrigTerm != nil {
				newOrigTerms = append(newOrigTerms, newOrigTerm)
			}
		} else {
			return nil, nil, false, nil
		}
	}

	return expression.NewOr(newTerms...), expression.NewOr(newOrigTerms...), isJoin, nil
}
