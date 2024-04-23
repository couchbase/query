//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"reflect"

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

/*
Base for ANY, EVERY, and ANY AND EVERY collection predicates.
*/
type CollPredicate interface {
	Expression
	Bindings() Bindings
	Satisfies() Expression
	EquivalentCollPred(other Expression) bool
}

type collPredBase struct {
	ExpressionBase
	bindings  Bindings
	satisfies Expression
	arrayId   int
}

func (this *collPredBase) PropagatesMissing() bool {
	return false
}

func (this *collPredBase) PropagatesNull() bool {
	return false
}

func (this *collPredBase) EquivalentTo(other Expression) bool {
	return this.equivalentTo(other, true)
}

func (this *collPredBase) EquivalentCollPred(other Expression) bool {
	return this.equivalentTo(other, false)
}

// strict = true: must be exactly the same
// strict = false: allow binding variable names to be different
func (this *collPredBase) equivalentTo(other Expression, strict bool) bool {
	if this.valueEquivalentTo(other) {
		return true
	}

	if reflect.TypeOf(this.expr) != reflect.TypeOf(other) {
		return false
	}

	o := other.(CollPredicate)
	if strict {
		return this.bindings.EquivalentTo(o.Bindings()) &&
			this.satisfies.EquivalentTo(o.Satisfies())
	}
	return equivalentBindingsWithExpression(this.bindings, o.Bindings(),
		Expressions{this.satisfies}, Expressions{o.Satisfies()})
}

func (this *collPredBase) coveredBy(keyspace string, exprs Expressions,
	options CoveredOptions) Covered {

	for _, expr := range exprs {
		if this.expr.EquivalentTo(expr) {
			return CoveredEquiv
		}
	}

	// if not checking binding vars (called from IsArrayCovered()), just call
	// CoveredBy() from ExpressionBase
	if !options.hasCoverBindVar() {
		return this.ExprBase().CoveredBy(keyspace, exprs, options)
	}

	// check binding expressions
	options.setCoverBindExpr()
	options.unsetCoverSatisfies()
	for _, b := range this.bindings {
		switch b.expr.CoveredBy(keyspace, exprs, options) {
		case CoveredFalse:
			return CoveredFalse
		}
	}

	// check satisfies expression
	options.unsetCoverBindExpr()
	options.setCoverSatisfies()
	allExprs := make(Expressions, 0, len(exprs))
	for _, expr := range exprs {
		if aexpr, ok := expr.(*All); ok {
			if array, aok := aexpr.array.(*Array); aok {
				if array.When() != nil {
					fc := make(map[Expression]value.Value, 4)
					fc = array.When().FilterExpressionCovers(fc)
					for e, _ := range fc {
						allExprs = append(allExprs, e)
					}
				}
				if fk, fok := array.valueMapping.(*FlattenKeys); fok {
					allExprs = append(allExprs, fk.Operands()...)
				} else {
					allExprs = append(allExprs, array.valueMapping)
				}
			} else {
				allExprs = append(allExprs, expr)
			}
		} else {
			allExprs = append(allExprs, expr)
		}

	}
	return this.satisfies.CoveredBy(keyspace, allExprs, options)
}

func (this *collPredBase) CoveredBy(keyspace string, exprs Expressions,
	options CoveredOptions) Covered {

	options.unsetCoverBindVar()
	return this.coveredBy(keyspace, exprs, options)
}

func (this *collPredBase) Children() Expressions {
	d := make(Expressions, 0, 1+len(this.bindings))

	for _, b := range this.bindings {
		d = append(d, b.Expression())
	}

	d = append(d, this.satisfies)
	return d
}

func (this *collPredBase) MapChildren(mapper Mapper) (err error) {
	err = this.bindings.MapExpressions(mapper)
	if err != nil {
		return
	}

	this.satisfies, err = mapper.Map(this.satisfies)
	if err != nil {
		return
	}

	return
}

func (this *collPredBase) SurvivesGrouping(groupKeys Expressions, allowed *value.ScopeValue) (
	bool, Expression) {
	for _, key := range groupKeys {
		if this.EquivalentTo(key) {
			return true, nil
		}
	}

	vars := _VARS_POOL.Get()
	defer _VARS_POOL.Put(vars)
	allowed = value.NewScopeValue(vars, allowed)
	allow_flags := value.NewValue(uint32(IDENT_IS_VARIABLE))
	for _, b := range this.bindings {
		allowed.SetField(b.Variable(), allow_flags)
	}

	for _, child := range this.Children() {
		ok, _ := child.SurvivesGrouping(groupKeys, allowed)
		if !ok {
			return ok, nil
		}
	}

	return true, nil
}

func (this *collPredBase) Bindings() Bindings {
	return this.bindings
}

func (this *collPredBase) Satisfies() Expression {
	return this.satisfies
}

func (this *collPredBase) ArrayId() int {
	return this.arrayId
}

func (this *collPredBase) SetArrayId(arrayId int) {
	this.arrayId = arrayId
}

var _VARS_POOL = util.NewStringInterfacePool(8)
