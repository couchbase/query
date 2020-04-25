//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
)

/*
 * Check whether an expression is null-rejecting for a particular keyspace.
 *
 * This is used for ANSI OUTER JOIN to ANSI INNER JOIN transformation, where the result
 * of an outer join may contain documents which the subservient side is MISSING (not joined),
 * however, if there exists a predicate in either the WHERE clause or an ON clause of an INNER JOIN
 * that can reject all such result documents, the outer join can be effectively converted to
 * an inner join.
 * Note in NoSQL it should really be called missing-rejecting, however null-rejecting is a
 * well-established term in SQL (since results of outer joins may contain null-extended rows),
 * we'll inherit this term.
 */
func nullRejExpr(chkNullRej *chkNullRej, alias string, expr expression.Expression) bool {
	chkNullRej.alias = alias
	chkNullRej.keyspaceNames = _KEYSPACE_NAMES_POOL.Get()
	defer func() {
		_KEYSPACE_NAMES_POOL.Put(chkNullRej.keyspaceNames)
		chkNullRej.keyspaceNames = nil
	}()
	chkNullRej.keyspaceNames[alias] = ""

	res, err := expr.Accept(chkNullRej)
	if err != nil {
		return false
	}

	return res.(bool)
}

type chkNullRej struct {
	alias         string
	keyspaceNames map[string]string
}

func newChkNullRej() *chkNullRej {
	return &chkNullRej{}
}

// Logic

func (this *chkNullRej) VisitAnd(expr *expression.And) (interface{}, error) {
	for _, op := range expr.Operands() {
		if op == nil {
			continue
		}

		// make sure the subterm references the keyspace
		if !expression.HasKeyspaceReferences(op, this.keyspaceNames) {
			continue
		}

		res, err := op.Accept(this)
		if err != nil {
			return false, err
		}

		nullRej := res.(bool)
		if nullRej {
			return true, nil
		}
	}

	return false, nil
}

func (this *chkNullRej) VisitOr(expr *expression.Or) (interface{}, error) {
	hasRef := false
	for _, op := range expr.Operands() {
		if op == nil {
			continue
		}

		// make sure the subterm references the keyspace
		if !expression.HasKeyspaceReferences(op, this.keyspaceNames) {
			return false, nil
		}

		res, err := op.Accept(this)
		if err != nil {
			return nil, err
		}

		nullRej := res.(bool)
		if !nullRej {
			return false, nil
		}
		hasRef = true
	}

	return hasRef, nil
}

func (this *chkNullRej) VisitNot(expr *expression.Not) (interface{}, error) {
	op := expr.Operand()
	res, err := op.Accept(this)
	if err != nil {
		return nil, err
	}

	nullRej := res.(bool)
	if !nullRej {
		return false, nil
	}

	switch op.(type) {
	case *expression.Eq, *expression.In, *expression.Within, *expression.Like, *expression.Between, *expression.And, *expression.Or:
		return true, nil
	}

	return false, nil
}

func (this *chkNullRej) visitOperands(args ...expression.Expression) (interface{}, error) {
	hasRef := false
	for _, arg := range args {
		if !expression.HasKeyspaceReferences(arg, this.keyspaceNames) {
			continue
		}

		res, err := arg.Accept(this)
		if err != nil {
			return nil, err
		}

		nullRej := res.(bool)
		if !nullRej {
			return false, nil
		}
		hasRef = true
	}

	return hasRef, nil
}

// Arithmetic

func (this *chkNullRej) VisitAdd(pred *expression.Add) (interface{}, error) {
	return this.visitOperands((pred.Operands())...)
}

func (this *chkNullRej) VisitDiv(pred *expression.Div) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

func (this *chkNullRej) VisitMod(pred *expression.Mod) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

func (this *chkNullRej) VisitMult(pred *expression.Mult) (interface{}, error) {
	return this.visitOperands((pred.Operands())...)
}

func (this *chkNullRej) VisitNeg(pred *expression.Neg) (interface{}, error) {
	return this.visitOperands(pred.Operand())
}

func (this *chkNullRej) VisitSub(pred *expression.Sub) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

// Case

func (this *chkNullRej) VisitSearchedCase(pred *expression.SearchedCase) (interface{}, error) {
	return false, nil
}

func (this *chkNullRej) VisitSimpleCase(pred *expression.SimpleCase) (interface{}, error) {
	return false, nil
}

// Collection

func (this *chkNullRej) visitAnyAndEvery(coll expression.CollectionPredicate) (interface{}, error) {
	keyspaceNames := this.keyspaceNames
	this.keyspaceNames = _KEYSPACE_NAMES_POOL.Get()
	defer func() {
		_KEYSPACE_NAMES_POOL.Put(this.keyspaceNames)
		this.keyspaceNames = keyspaceNames
	}()
	this.keyspaceNames[this.alias] = ""

	aliasIdent := expression.NewIdentifier(this.alias)

	for _, binding := range coll.Bindings() {
		if binding.Expression().DependsOn(aliasIdent) {
			this.keyspaceNames[binding.Variable()] = ""
		}
	}

	if expression.HasKeyspaceReferences(coll.Satisfies(), this.keyspaceNames) {
		return coll.Satisfies().Accept(this)
	}

	return false, nil
}

func (this *chkNullRej) VisitAny(pred *expression.Any) (interface{}, error) {
	return this.visitAnyAndEvery(pred)
}

func (this *chkNullRej) VisitEvery(pred *expression.Every) (interface{}, error) {
	return this.visitAnyAndEvery(pred)
}

func (this *chkNullRej) VisitAnyEvery(pred *expression.AnyEvery) (interface{}, error) {
	return this.visitAnyAndEvery(pred)
}

func (this *chkNullRej) VisitArray(pred *expression.Array) (interface{}, error) {
	return false, nil
}

func (this *chkNullRej) VisitFirst(pred *expression.First) (interface{}, error) {
	return false, nil
}

func (this *chkNullRej) VisitObject(pred *expression.Object) (interface{}, error) {
	return false, nil
}

func (this *chkNullRej) VisitExists(pred *expression.Exists) (interface{}, error) {
	return pred.Operand().Accept(this)
}

/* IN, WITHIN expressions are null rejecting */
func (this *chkNullRej) VisitIn(pred *expression.In) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

func (this *chkNullRej) VisitWithin(pred *expression.Within) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

// Comparison

/* all relational comparison operations are null rejecting */
func (this *chkNullRej) VisitBetween(pred *expression.Between) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

func (this *chkNullRej) VisitEq(pred *expression.Eq) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

func (this *chkNullRej) VisitLE(pred *expression.LE) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

func (this *chkNullRej) VisitLike(pred *expression.Like) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

func (this *chkNullRej) VisitLT(pred *expression.LT) (interface{}, error) {
	return this.visitOperands(pred.First(), pred.Second())
}

func (this *chkNullRej) VisitIsMissing(pred *expression.IsMissing) (interface{}, error) {
	return false, nil
}

/* IS NOT MISSING is null rejecting */
func (this *chkNullRej) VisitIsNotMissing(pred *expression.IsNotMissing) (interface{}, error) {
	return this.visitOperands(pred.Operand())
}

/* IS NOT NULL is null rejecting */
func (this *chkNullRej) VisitIsNotNull(pred *expression.IsNotNull) (interface{}, error) {
	return this.visitOperands(pred.Operand())
}

func (this *chkNullRej) VisitIsNotValued(pred *expression.IsNotValued) (interface{}, error) {
	return false, nil
}

func (this *chkNullRej) VisitIsNull(pred *expression.IsNull) (interface{}, error) {
	return false, nil
}

/* IS VALUED is null rejecting */
func (this *chkNullRej) VisitIsValued(pred *expression.IsValued) (interface{}, error) {
	return this.visitOperands(pred.Operand())
}

// Concat
func (this *chkNullRej) VisitConcat(pred *expression.Concat) (interface{}, error) {
	return this.visitOperands((pred.Operands())...)
}

// Constant
func (this *chkNullRej) VisitConstant(pred *expression.Constant) (interface{}, error) {
	return false, nil
}

// Identifier
func (this *chkNullRej) VisitIdentifier(pred *expression.Identifier) (interface{}, error) {
	_, ok := this.keyspaceNames[pred.Identifier()]
	return ok, nil
}

// Construction

func (this *chkNullRej) VisitArrayConstruct(pred *expression.ArrayConstruct) (interface{}, error) {
	return false, nil
}

func (this *chkNullRej) VisitObjectConstruct(pred *expression.ObjectConstruct) (interface{}, error) {
	return false, nil
}

// Navigation

func (this *chkNullRej) VisitElement(pred *expression.Element) (interface{}, error) {
	return false, nil
}

func (this *chkNullRej) VisitField(pred *expression.Field) (interface{}, error) {
	return this.visitOperands(pred.First())
}

func (this *chkNullRej) VisitFieldName(pred *expression.FieldName) (interface{}, error) {
	return false, nil
}

func (this *chkNullRej) VisitSlice(pred *expression.Slice) (interface{}, error) {
	return false, nil
}

// Self
func (this *chkNullRej) VisitSelf(pred *expression.Self) (interface{}, error) {
	return false, nil
}

// Function
func (this *chkNullRej) VisitFunction(pred expression.Function) (interface{}, error) {
	return false, nil
}

// Subquery
func (this *chkNullRej) VisitSubquery(pred expression.Subquery) (interface{}, error) {
	return false, nil
}

// NamedParameter
func (this *chkNullRej) VisitNamedParameter(pred expression.NamedParameter) (interface{}, error) {
	return false, nil
}

// PositionalParameter
func (this *chkNullRej) VisitPositionalParameter(pred expression.PositionalParameter) (interface{}, error) {
	return false, nil
}

// Cover
func (this *chkNullRej) VisitCover(pred *expression.Cover) (interface{}, error) {
	return false, nil
}

// All
func (this *chkNullRej) VisitAll(pred *expression.All) (interface{}, error) {
	return false, nil
}

var _KEYSPACE_NAMES_POOL = util.NewStringStringPool(16)
