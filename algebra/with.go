//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type CycleCheck struct {
	cycle expression.Expressions
}

func NewCycleCheck(cycle expression.Expressions) *CycleCheck {
	return &CycleCheck{
		cycle: cycle,
	}
}

func (this *CycleCheck) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 1)

	r["cycle"] = this.cycle
	return json.Marshal(r)
}

type With struct {
	alias        string
	expr         expression.Expression
	rexpr        expression.Expression
	isUnion      bool
	config       value.Value
	cycle        *CycleCheck
	errorContext expression.ErrorContext
}

func NewWith(alias string, expr, rexpr expression.Expression, isUnion bool, config value.Value, cycle *CycleCheck) expression.With {
	return &With{
		alias:   alias,
		expr:    expr,
		rexpr:   rexpr,
		isUnion: isUnion,
		config:  config,
		cycle:   cycle,
	}
}

func (this *With) Alias() string {
	return this.alias
}

func (this *With) Expression() expression.Expression {
	return this.expr
}

func (this *With) SetExpression(expr expression.Expression) {
	this.expr = expr
}

func (this *With) RecursiveExpression() expression.Expression {
	return this.rexpr
}

func (this *With) SetRecursiveExpression(rexpr expression.Expression) {
	this.rexpr = rexpr
}

func (this *With) IsRecursive() bool {
	return this.rexpr != nil
}

func (this *With) SetUnion() {
	this.isUnion = true
}

func (this *With) IsUnion() bool {
	return this.isUnion
}

func (this *With) Config() value.Value {
	return this.config
}

func (this *With) CycleFields() expression.Expressions {
	if this.cycle != nil {
		return this.cycle.cycle
	}
	return nil
}

func (this *With) ErrorContext() string {
	return this.errorContext.String()
}

func (this *With) SetErrorContext(line int, column int) {
	this.errorContext.Set(line, column)
}

/*
Representation as a N1QL string
*/
func (this *With) String() string {
	s := "`" + this.alias + "`" + " AS ( "
	s += this.expr.String()
	s += " ) "

	return s
}

func (this *With) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 2)
	r["alias"] = this.alias
	r["expr"] = this.expr.String()
	if this.rexpr != nil {
		r["rexpr"] = this.rexpr.String()
		if this.isUnion {
			r["is_union"] = true
		}
		if this.config != nil {
			r["config"] = this.config
		}
		if this.cycle != nil {
			r["cycle"] = this.cycle
		}
	}
	return json.Marshal(r)
}

/*
perform split on top-level Union/ Union All
and set new anchor_clause and recursive_clause

error if recursive alias is referenced in anchor clause
*/
func (this *With) SplitRecursive() error {

	aSubq, ok := this.expr.(*Subquery)
	if !ok {
		// couldn't cast to algebra
		// handles case when not a subquery as well
		return nil
	}

	var isUnion bool
	var first, second Subresult
	switch res := aSubq.query.subresult.(type) {
	case *Union:
		isUnion = true
		first = res.first
		second = res.second
	case *UnionAll:
		first = res.first
		second = res.second
	}

	if first == nil || second == nil {
		// only 1 arm
		return nil
	}

	var found bool
	var err error
	found, err = checkRecursive(this.alias, first)
	// recursive reference not allowed in anchor expression
	if err != nil {
		return err
	}

	if found {
		return errors.NewRecursiveAnchorError("FROM expression", this.alias, "semantics.fromExpr.recursive_anchor")
	}

	found, err = checkRecursive(this.alias, second)
	if err != nil {
		return err
	}

	if !found {
		// no split, reject recursive hint
		return nil
	}

	if aSubq.query.order != nil || aSubq.query.limit != nil || aSubq.query.offset != nil {
		return errors.NewRecursiveWithSemanticError("Order/Limit/Offset not allowed")
	}

	var firstSelect, secondSelect *Select
	if firstSelectTerm, ok := first.(*SelectTerm); ok {
		// order/limit/offset check are handled in visitSelect is using Select Term
		firstSelect = firstSelectTerm.query
	} else {
		firstSelect = NewSelect(first, nil, nil, nil, nil)
	}

	if secondSelectTerm, ok := second.(*SelectTerm); ok {
		secondSelect = secondSelectTerm.query
	} else {
		// same as firstSelect for secondSelect when a subselect or subquery
		secondSelect = NewSelect(second, nil, nil, nil, nil)
	}
	secondSelect.SetRecursiveWith(true)

	firstSubq := NewSubquery(firstSelect)
	secondSubq := NewSubquery(secondSelect)

	this.SetExpression(firstSubq)
	this.SetRecursiveExpression(secondSubq)
	if isUnion {
		this.SetUnion()
	}
	return nil
}

type WithClause struct {
	withs     expression.Withs
	recursive bool
}

func NewWithClause(recursive bool, withs expression.Withs) *WithClause {
	return &WithClause{
		recursive: recursive,
		withs:     withs,
	}
}

func (this *WithClause) Bindings() expression.Withs {
	return this.withs
}

func (this *WithClause) SetBindings(withs expression.Withs) {
	this.withs = withs
}

func (this *WithClause) IsRecursive() bool {
	return this.recursive
}

func (this *WithClause) SetRecursive() {
	this.recursive = true
}

func (this *WithClause) Expressions() expression.Expressions {
	return this.withs.Expressions()
}

func (this *WithClause) MapExpressions(mapper expression.Mapper) error {
	return this.withs.MapExpressions(mapper)
}
