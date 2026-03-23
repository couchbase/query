//  Copyright 2025-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type planContext struct {
	expression.MapperBase

	withs     *value.ScopeValue
	vars      *value.ScopeValue
	keyspaces *value.ScopeValue

	// for subquery plans in prepared
	subqMap   map[string]*subqPlanInfo
	subqPlans *algebra.SubqueryPlans
}

func newPlanContext(subqMap map[string]*subqPlanInfo, subqPlans *algebra.SubqueryPlans) *planContext {
	rv := &planContext{
		withs:     value.NewScopeValue(make(map[string]interface{}), nil),
		vars:      value.NewScopeValue(make(map[string]interface{}), nil),
		keyspaces: value.NewScopeValue(make(map[string]interface{}), nil),
		subqMap:   subqMap,
		subqPlans: subqPlans,
	}

	rv.SetMapper(rv)
	return rv
}

func subqPlanContext(parent *planContext) *planContext {
	// parent should not be nil
	if parent == nil {
		panic("subqPlanContext: parent pointer is nil")
	}

	// the subqMap and subqPlans should contain subqueries at all levels, and it's ok to share
	// the pointers since this is currently only used for unmarshal of subquery plans from a
	// prepared statement (no parallelism)
	rv := &planContext{
		withs:     value.NewScopeValue(make(map[string]interface{}), parent.withs),
		vars:      value.NewScopeValue(make(map[string]interface{}), parent.vars),
		keyspaces: value.NewScopeValue(make(map[string]interface{}), parent.keyspaces),
		subqMap:   parent.subqMap,
		subqPlans: parent.subqPlans,
	}

	rv.SetMapper(rv)
	return rv
}

func (this *planContext) VisitAny(expr *expression.Any) (interface{}, error) {
	err := this.pushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}
	defer this.popBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *planContext) VisitEvery(expr *expression.Every) (interface{}, error) {
	err := this.pushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}
	defer this.popBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *planContext) VisitAnyEvery(expr *expression.AnyEvery) (interface{}, error) {
	err := this.pushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}
	defer this.popBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *planContext) VisitArray(expr *expression.Array) (interface{}, error) {
	err := this.pushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}
	defer this.popBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *planContext) VisitFirst(expr *expression.First) (interface{}, error) {
	err := this.pushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}
	defer this.popBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *planContext) VisitObject(expr *expression.Object) (interface{}, error) {
	err := this.pushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}
	defer this.popBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *planContext) pushBindings(bindings expression.Bindings, push bool) (err error) {
	vars := this.vars
	keyspaces := this.keyspaces

	if push {
		vars = value.NewScopeValue(make(map[string]interface{}, len(bindings)), vars)
		keyspaces = value.NewScopeValue(make(map[string]interface{}, len(bindings)), keyspaces)
	}

	for _, b := range bindings {
		_, err = this.Map(b.Expression())
		if err != nil {
			return
		}
		// no need to call b.SetExpression() since we don't expect expr to change, the
		// only changes should be flags on identifiers

		variable := b.Variable()
		ident_flags := uint32(expression.IDENT_IS_VARIABLE)
		if b.Static() {
			ident_flags |= expression.IDENT_IS_STATIC_VAR
		}
		if b.FuncVariable() {
			ident_flags |= expression.IDENT_IS_FUNC_VAR
		}
		ident_val := value.NewValue(ident_flags)
		vars.SetField(variable, ident_val)

		if b.NameVariable() != "" {
			variable = b.NameVariable()
			ident_flags := uint32(expression.IDENT_IS_VARIABLE)
			ident_val := value.NewValue(ident_flags)
			vars.SetField(variable, ident_val)
		}
	}

	if push {
		this.vars = vars
		this.keyspaces = keyspaces
	}

	return
}

func (this *planContext) popBindings() {
	this.vars = this.vars.Parent().(*value.ScopeValue)
	this.keyspaces = this.keyspaces.Parent().(*value.ScopeValue)
}

func (this *planContext) addWiths(withs expression.Withs) (err error) {
	for _, with := range withs {
		_, err = this.Map(with.Expression())
		if err != nil {
			return
		}
		// no need to call with.SetExpression() since we don't expect expr to change, the
		// only changes should be flags on identifiers

		variable := with.Alias()
		ident_flags := uint32(expression.IDENT_IS_WITH_ALIAS | expression.IDENT_IS_STATIC_VAR)
		ident_val := value.NewValue(ident_flags)
		this.withs.SetField(variable, ident_val)
	}

	return
}

func (this *planContext) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	ident := expr.Identifier()
	if _, ok := this.withs.Field(ident); ok {
		expr.SetWithAlias(true)
	} else if _, ok := this.vars.Field(ident); ok {
		expr.SetBindingVariable(true)
	} else if ident_val, ok := this.keyspaces.Field(ident); ok {
		ident_flags := uint32(ident_val.ActualForIndex().(int64))
		if (ident_flags & expression.IDENT_IS_KEYSPACE) != 0 {
			expr.SetKeyspaceAlias(true)
		}
		if (ident_flags & expression.IDENT_IS_UNNEST_ALIAS) != 0 {
			expr.SetUnnestAlias(true)
		}
		if (ident_flags & expression.IDENT_IS_EXPR_TERM) != 0 {
			expr.SetExprTermAlias(true)
		}
		if (ident_flags & expression.IDENT_IS_SUBQ_TERM) != 0 {
			expr.SetSubqTermAlias(true)
		}
	}
	return expr, nil
}

func (this *planContext) VisitSubquery(expr expression.Subquery) (interface{}, error) {
	// use a new planContext for an extra scope
	planContext := subqPlanContext(this)

	err := expr.MapChildren(planContext)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *planContext) addKeyspaceAlias(alias string) {
	if _, ok := this.keyspaces.Field(alias); !ok {
		ident_flags := uint32(expression.IDENT_IS_KEYSPACE)
		this.keyspaces.SetField(alias, value.NewValue(ident_flags))
	}
}

func (this *planContext) addUnnestAlias(alias string) {
	if _, ok := this.keyspaces.Field(alias); !ok {
		ident_flags := uint32(expression.IDENT_IS_KEYSPACE | expression.IDENT_IS_UNNEST_ALIAS)
		this.keyspaces.SetField(alias, value.NewValue(ident_flags))
	}
}

func (this *planContext) addExprTermAlias(alias string) {
	if _, ok := this.keyspaces.Field(alias); !ok {
		ident_flags := uint32(expression.IDENT_IS_KEYSPACE | expression.IDENT_IS_EXPR_TERM)
		this.keyspaces.SetField(alias, value.NewValue(ident_flags))
	}
}

func (this *planContext) addSubqTermAlias(alias string) {
	if _, ok := this.keyspaces.Field(alias); !ok {
		ident_flags := uint32(expression.IDENT_IS_KEYSPACE | expression.IDENT_IS_SUBQ_TERM)
		this.keyspaces.SetField(alias, value.NewValue(ident_flags))
	}
}

// functionality for handling subquery map

type subqPlanInfo struct {
	op   []byte
	isks map[string]bool
}

func (this *planContext) hasSubqueryMap() bool {
	return this.subqMap != nil
}

// Look for subquery expressions in the expr just parsed, and for each subquery, match with the
// subquery plans from prepared statement; when matched, unmarshal the subquery plan and put the
// subquery plan into algebra.SubqueryPlans (for the prepared statement)
func (this *planContext) checkSubqueryMap(expr expression.Expression) (expression.Expression, error) {
	if len(this.subqMap) == 0 {
		return expr, nil
	}

	subqs, err := expression.ListSubqueries(expression.Expressions{expr}, false)
	if err != nil {
		return expr, err
	}

	for _, subqExpr := range subqs {
		subq, _ := subqExpr.(*algebra.Subquery)
		str := subq.String()
		if planInfo, ok := this.subqMap[str]; ok {
			// unmarshal the subquery plan here (in case there are identical subqueries
			// present we need separate plan for each subquery)
			var op_type struct {
				Operator string `json:"#operator"`
			}

			err = json.Unmarshal(planInfo.op, &op_type)
			if err != nil {
				return nil, err
			}

			subPlanContext := subqPlanContext(this)
			subqPlan, err := MakeOperator(op_type.Operator, planInfo.op, subPlanContext)
			if err != nil {
				return nil, err
			}
			// no need for locking since this is only called under unmarshal (no parallelism)
			this.subqPlans.Set(subq.Select(), nil, NewQueryPlan(subqPlan), planInfo.isks, false)
		}
	}
	return expr, nil
}
