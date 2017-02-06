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
	"github.com/couchbase/query/expression"
)

// Converts the predicate to use a dynamic index
func DynamicFor(pred expression.Expression, variable *expression.Identifier, pairs *expression.Pairs) (
	expr expression.Expression, err error) {

	pred = pred.Copy()
	pred, err = Fold(pred)
	if err != nil {
		return nil, err
	}

	dyn := newDynamic(variable, pairs)
	rv, err := pred.Accept(dyn)
	if err != nil {
		return nil, err
	}

	return rv.(expression.Expression), nil
}

type dynamic struct {
	expression.MapperBase

	variable *expression.Identifier
	pairs    *expression.Pairs
}

func newDynamic(variable *expression.Identifier, pairs *expression.Pairs) *dynamic {
	rv := &dynamic{
		variable: variable,
		pairs:    pairs,
	}

	rv.SetMapper(rv)
	rv.SetMapFunc(
		func(expr expression.Expression) (expression.Expression, error) {
			alias := expr.Alias()
			if alias == "" {
				return expr, nil
			}

			sat := expression.NewAnd(
				expression.NewGE(
					rv.NewVariable(),
					rv.NewArray(alias, true),
				),
				expression.NewLT(
					rv.NewVariable(),
					rv.NewArray(expression.NewSuccessor(expression.NewConstant(alias))),
				),
			)
			any := expression.NewAny(rv.NewBindings(), sat)
			return expression.NewAnd(expr, any), nil
		})

	return rv
}

// Collection

func (this *dynamic) VisitAny(expr *expression.Any) (rv interface{}, err error) {
	rsat := expr.Satisfies().Copy()

	for _, binding := range expr.Bindings() {
		id := expression.NewIdentifier(binding.Variable())
		replacer := expression.NewReplacer(id, binding.Expression())
		rsat, err = replacer.Map(rsat)
		if err != nil {
			return nil, err
		}
	}

	return rsat.Accept(this)
}

func (this *dynamic) VisitAnyEvery(expr *expression.AnyEvery) (rv interface{}, err error) {
	rsat := expr.Satisfies().Copy()

	for _, binding := range expr.Bindings() {
		id := expression.NewIdentifier(binding.Variable())
		replacer := expression.NewReplacer(id, binding.Expression())
		rsat, err = replacer.Map(rsat)
		if err != nil {
			return nil, err
		}
	}

	return rsat.Accept(this)
}

func (this *dynamic) VisitExists(expr *expression.Exists) (interface{}, error) {
	alias := expr.Operand().Alias()
	if alias == "" {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGT(
			this.NewVariable(),
			expression.EMPTY_ARRAY_EXPR,
		),
		expression.NewLT(
			this.NewVariable(),
			_EMPTY_OBJECT_EXPR,
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitIn(expr *expression.In) (interface{}, error) {
	alias := expr.First().Alias()
	cons, ok := expr.Second().(*expression.ArrayConstruct)

	if alias == "" || !ok {
		return expr, nil
	}

	statics := make([]expression.Expression, len(cons.Operands()))
	for i, op := range cons.Operands() {
		statics[i] = op.Static()
		if statics[i] == nil {
			return expr, nil
		}
	}

	pairs := make([]expression.Expression, len(statics))
	for i, s := range statics {
		pairs[i] = this.NewArray(alias, s)
	}

	sat := expression.NewIn(
		this.NewVariable(),
		expression.NewArrayConstruct(pairs...),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

// Comparison

func (this *dynamic) VisitBetween(expr *expression.Between) (interface{}, error) {
	alias := expr.First().Alias()
	low := expr.Second().Static()
	high := expr.Third().Static()

	if alias == "" || low == nil || high == nil {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGE(
			this.NewVariable(),
			this.NewArray(alias, low),
		),
		expression.NewLE(
			this.NewVariable(),
			this.NewArray(alias, high),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitEq(expr *expression.Eq) (interface{}, error) {
	alias := expr.First().Alias()
	static := expr.Second().Static()

	if alias == "" || static == nil {
		alias = expr.Second().Alias()
		static = expr.First().Static()
	}

	if alias == "" || static == nil {
		return expr, nil
	}

	sat := expression.NewEq(
		this.NewVariable(),
		this.NewArray(alias, static),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitLE(expr *expression.LE) (interface{}, error) {
	alias := expr.First().Alias()
	static := expr.Second().Static()

	if alias == "" || static == nil {
		alias = expr.Second().Alias()
		static = expr.First().Static()
		if alias != "" && static != nil {
			return this.visitGE(expr, alias, static)
		}
	}

	if alias == "" || static == nil {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGE(
			this.NewVariable(),
			this.NewArray(alias, false),
		),
		expression.NewLE(
			this.NewVariable(),
			this.NewArray(alias, static),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) visitGE(expr *expression.LE, alias string, static expression.Expression) (
	interface{}, error) {

	sat := expression.NewAnd(
		expression.NewGE(
			this.NewVariable(),
			this.NewArray(alias, static),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(expression.NewSuccessor(expression.NewConstant(alias))),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitLike(expr *expression.Like) (interface{}, error) {
	alias := expr.First().Alias()
	static := expr.Second().Static()

	if alias == "" || static == nil {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGE(
			this.NewVariable(),
			this.NewArray(alias, expression.NewLikePrefix(static)),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(alias, expression.NewLikeStop(static)),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitLT(expr *expression.LT) (interface{}, error) {
	alias := expr.First().Alias()
	static := expr.Second().Static()

	if alias == "" || static == nil {
		alias = expr.Second().Alias()
		static = expr.First().Static()
		if alias != "" && static != nil {
			return this.visitGT(expr, alias, static)
		}
	}

	if alias == "" || static == nil {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGE(
			this.NewVariable(),
			this.NewArray(alias, false),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(alias, static),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) visitGT(expr *expression.LT, alias string, static expression.Expression) (
	interface{}, error) {

	sat := expression.NewAnd(
		expression.NewGT(
			this.NewVariable(),
			this.NewArray(alias, static),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(expression.NewSuccessor(expression.NewConstant(alias))),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitIsNotMissing(expr *expression.IsNotMissing) (interface{}, error) {
	alias := expr.Operand().Alias()
	if alias == "" {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGT(
			this.NewVariable(),
			this.NewArray(alias),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(expression.NewSuccessor(expression.NewConstant(alias))),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitIsNotNull(expr *expression.IsNotNull) (interface{}, error) {
	alias := expr.Operand().Alias()
	if alias == "" {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGE(
			this.NewVariable(),
			this.NewArray(alias, false),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(expression.NewSuccessor(expression.NewConstant(alias))),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	alias := expr.Operand().Alias()
	if alias == "" {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGT(
			this.NewVariable(),
			this.NewArray(alias),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(alias, false),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

func (this *dynamic) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	alias := expr.Operand().Alias()
	if alias == "" {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGE(
			this.NewVariable(),
			this.NewArray(alias, false),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(expression.NewSuccessor(expression.NewConstant(alias))),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

// Logic

func (this *dynamic) VisitAnd(expr *expression.And) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *dynamic) VisitOr(expr *expression.Or) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *dynamic) VisitFunction(expr expression.Function) (interface{}, error) {
	switch expr := expr.(type) {
	case *expression.RegexpLike:
		return this.visitRegexpLike(expr)
	default:
		return expr, nil
	}
}

func (this *dynamic) visitRegexpLike(expr *expression.RegexpLike) (interface{}, error) {
	alias := expr.First().Alias()
	static := expr.Second().Static()

	if alias == "" || static == nil {
		return expr, nil
	}

	sat := expression.NewAnd(
		expression.NewGE(
			this.NewVariable(),
			this.NewArray(alias, expression.NewRegexpPrefix(static)),
		),
		expression.NewLT(
			this.NewVariable(),
			this.NewArray(alias, expression.NewRegexpStop(static)),
		),
	)
	any := expression.NewAny(this.NewBindings(), sat)
	return expression.NewAnd(expr, any), nil
}

// Internal

func (this *dynamic) NewVariable() *expression.Identifier {
	return this.variable
}

func (this *dynamic) NewBindings() expression.Bindings {
	binding := expression.NewSimpleBinding(
		this.variable.Identifier(),
		this.pairs,
	)
	return expression.Bindings{binding}
}

func (this *dynamic) NewArray(items ...interface{}) expression.Expression {
	exprs := make([]expression.Expression, len(items))
	for i, item := range items {
		switch item := item.(type) {
		case expression.Expression:
			exprs[i] = item
		default:
			exprs[i] = expression.NewConstant(item)
		}
	}

	return expression.NewArrayConstruct(exprs...)
}
