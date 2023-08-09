//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	keyspaceNames map[string]string, isOnclause, doSelec, advisorValidate bool,
	context *PrepareContext) (expression.Expression, error) {

	if len(baseKeyspaces) == 0 {
		return nil, errors.NewPlanError(nil, "ClassifyExpr: invalid argument baseKeyspaces")
	}

	// make sure document count is available
	for _, baseKeyspace := range baseKeyspaces {
		keyspace := baseKeyspace.Keyspace()
		if keyspace != "" && !baseKeyspace.HasDocCount() {
			baseKeyspace.SetDocCount(optDocCount(keyspace))
			baseKeyspace.SetHasDocCount()
		}
	}

	return ClassifyExprKeyspace(expr, baseKeyspaces, keyspaceNames, "", isOnclause, doSelec, advisorValidate, context)
}

func ClassifyExprKeyspace(expr expression.Expression, baseKeyspaces map[string]*base.BaseKeyspace,
	keyspaceNames map[string]string, alias string, isOnclause, doSelec, advisorValidate bool,
	context *PrepareContext) (expression.Expression, error) {

	classifier := newExprClassifier(baseKeyspaces, keyspaceNames, alias, isOnclause,
		doSelec, advisorValidate, context)
	_, err := expr.Accept(classifier)
	if err != nil {
		return nil, err
	}

	if doSelec {
		optCheckRangeExprs(baseKeyspaces, advisorValidate, context)
	}

	return classifier.extraExpr, nil
}

type exprClassifier struct {
	baseKeyspaces   map[string]*base.BaseKeyspace
	keyspaceNames   map[string]string
	alias           string
	recursion       bool
	recurseExpr     expression.Expression
	recursionJoin   bool
	recurseJoinExpr expression.Expression
	isOnclause      bool
	extraExpr       expression.Expression
	doSelec         bool
	advisorValidate bool
	context         *PrepareContext
}

func newExprClassifier(baseKeyspaces map[string]*base.BaseKeyspace, keyspaceNames map[string]string,
	alias string, isOnclause, doSelec, advisorValidate bool, context *PrepareContext) *exprClassifier {

	return &exprClassifier{
		baseKeyspaces:   baseKeyspaces,
		keyspaceNames:   keyspaceNames,
		alias:           alias,
		isOnclause:      isOnclause,
		doSelec:         doSelec,
		advisorValidate: advisorValidate,
		context:         context,
	}
}

func (this *exprClassifier) addConstant(expr expression.Expression) {
	if this.extraExpr == nil {
		this.extraExpr = expr
	} else {
		this.extraExpr = expression.NewAnd(this.extraExpr, expr)
	}
}

func (this *exprClassifier) VisitAnd(expr *expression.And) (interface{}, error) {

	var err error
	for _, op := range expr.Operands() {
		switch op := op.(type) {
		case *expression.And:
			_, err = this.VisitAnd(op)
		case *expression.Or:
			_, err = this.VisitOr(op)
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
		this.addConstant(expr)
		return expression.TRUE_EXPR, nil
	}

	newExpr := false
	orTerms := make(expression.Expressions, 0, len(or.Operands()))

	posParams := this.context.PositionalArgs()
	namedParams := this.context.NamedArgs()

	for _, op := range or.Operands() {
		skip := false
		var cop value.Value

		// replace named/pos param thus eliminating unwanted span generation for cases like ($c=1)
		if len(posParams) > 0 || len(namedParams) > 0 {
			rop, err := base.ReplaceParameters(op, namedParams, posParams)
			if err != nil {
				return nil, err
			}
			cop = rop.Value()
		} else {
			cop = op.Value()
		}

		if op.HasExprFlag(expression.EXPR_VALUE_MISSING) || op.HasExprFlag(expression.EXPR_VALUE_NULL) {
			skip = true
		} else if cop != nil {
			// MB-58201: check for true subterm after replacing parameters
			if cop.Truth() {
				this.addConstant(expr)
				return expression.TRUE_EXPR, nil
			}

			// FALSE subterm can be skipped
			skip = true
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
			this.addConstant(expr)
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
	keyspaces, err := expression.CountKeySpaces(expr, this.keyspaceNames)
	if err != nil {
		return nil, err
	}

	if len(keyspaces) < 1 {
		// remember filters that do not reference any keyspace
		this.addConstant(expr)
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

	isOuter := false
	if this.isOnclause {
		// remove references to keyspaces that's already processed
		for kspace, _ := range keyspaces {
			if baseKspace, ok := this.baseKeyspaces[kspace]; ok {
				if baseKspace.PlanDone() {
					delete(keyspaces, kspace)
				}
			}
		}
		if this.alias != "" {
			if baseKspace, ok := this.baseKeyspaces[this.alias]; ok && baseKspace.IsOuter() {
				isOuter = true
			}
		}
	}

	optBits := int32(0)
	if this.doSelec {
		optBits = getOptBits(this.baseKeyspaces, origKeyspaces)
	}

	for kspace, _ := range keyspaces {
		baseKspace, ok := this.baseKeyspaces[kspace]
		if !ok {
			return nil, errors.NewPlanInternalError(fmt.Sprintf("exprClassifier.visitDefault: missing keyspace %s", kspace))
		}

		notPushable := false
		if this.isOnclause {
			if baseKspace.PlanDone() {
				// skip keyspaces already processed
				continue
			} else if this.alias != "" && kspace != this.alias {
				if baseKspace.IsOuter() && !isOuter {
					// Predicate from ON-clause of an inner join, but referencing
					// an outer keyspace, is non-pushable and needs to be added
					// to the outer keyspace such that it can be evaluated later
					// as post-join filter for the outer keyspace during join
					// enumeration. (not an issue without join enumeration).
					notPushable = true
				} else {
					// if alias is set, only add filter to the specified keyspace
					continue
				}
			}
		}

		filter := base.NewFilter(dnfExpr, origExpr, keyspaces, origKeyspaces,
			this.isOnclause, len(origKeyspaces) > 1)
		if this.doSelec && !baseKspace.IsUnnest() && baseKspace.DocCount() >= 0 {
			optFilterSelectivity(filter, this.advisorValidate, this.context)
			filter.SetOptBits(optBits)
		}
		if len(subqueries) > 0 {
			filter.SetSubq()
		}
		if notPushable {
			filter.SetNotPushable()
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
					if this.doSelec && !baseKspace.IsUnnest() && baseKspace.DocCount() >= 0 {
						optFilterSelectivity(newFilter, this.advisorValidate, this.context)
						newFilter.SetOptBits(baseKspace.OptBit())
					}
					baseKspace.AddFilter(newFilter)
				}
			}
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
		_, err := ClassifyExprKeyspace(op, baseKeyspaces, this.keyspaceNames, this.alias,
			this.isOnclause, this.doSelec, this.advisorValidate, this.context)
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
