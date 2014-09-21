//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"fmt"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type Formalizer struct {
	Allowed  value.Value
	Keyspace string
}

func NewFormalizer() *Formalizer {
	return &Formalizer{
		Allowed: value.NewValue(make(map[string]interface{})),
	}
}

func (this *Formalizer) Map(expr expression.Expression) (expression.Expression, error) {
	exp, err := expr.Accept(this)
	if err != nil {
		return nil, err
	}

	return exp.(expression.Expression), nil
}

func (this *Formalizer) MapBindings() bool { return false }

// Arithmetic

func (this *Formalizer) VisitAdd(expr *expression.Add) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitDiv(expr *expression.Div) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitMod(expr *expression.Mod) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitMult(expr *expression.Mult) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitNeg(expr *expression.Neg) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitSub(expr *expression.Sub) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Case

func (this *Formalizer) VisitSearchedCase(expr *expression.SearchedCase) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitSimpleCase(expr *expression.SimpleCase) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Collection

func (this *Formalizer) VisitAny(expr *expression.Any) (interface{}, error) {
	defer this.PopBindings()
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitArray(expr *expression.Array) (interface{}, error) {
	defer this.PopBindings()
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitEvery(expr *expression.Every) (interface{}, error) {
	defer this.PopBindings()
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitExists(expr *expression.Exists) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitFirst(expr *expression.First) (interface{}, error) {
	defer this.PopBindings()
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitIn(expr *expression.In) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitWithin(expr *expression.Within) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Comparison

func (this *Formalizer) VisitBetween(expr *expression.Between) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitEq(expr *expression.Eq) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitLE(expr *expression.LE) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitLike(expr *expression.Like) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitLT(expr *expression.LT) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitIsMissing(expr *expression.IsMissing) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitIsNull(expr *expression.IsNull) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitIsValued(expr *expression.IsValued) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Concat

func (this *Formalizer) VisitConcat(expr *expression.Concat) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Constant

func (this *Formalizer) VisitConstant(expr *expression.Constant) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Identifier

func (this *Formalizer) VisitIdentifier(expr *expression.Identifier) (interface{}, error) {
	_, ok := this.Allowed.Field(expr.Identifier())
	if ok {
		return expr, nil
	}

	if this.Keyspace == "" {
		return nil, fmt.Errorf("Ambiguous reference to field %v.", expr.Identifier())
	}

	return expression.NewField(
			expression.NewIdentifier(this.Keyspace),
			expression.NewFieldName(expr.Identifier())),
		nil
}

// Construction

func (this *Formalizer) VisitArrayConstruct(expr *expression.ArrayConstruct) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitObjectConstruct(expr *expression.ObjectConstruct) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Logic

func (this *Formalizer) VisitAnd(expr *expression.And) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitNot(expr *expression.Not) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitOr(expr *expression.Or) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Navigation

func (this *Formalizer) VisitElement(expr *expression.Element) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitField(expr *expression.Field) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitFieldName(expr *expression.FieldName) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitSlice(expr *expression.Slice) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Function

func (this *Formalizer) VisitFunction(expr expression.Function) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Bindings

func (this *Formalizer) PushBindings(bindings expression.Bindings) (err error) {
	this.Allowed = value.NewScopeValue(make(map[string]interface{}, len(bindings)), this.Allowed)

	for _, b := range bindings {
		_, ok := this.Allowed.Field(b.Variable())
		if ok {
			return errors.NewError(nil,
				fmt.Sprintf("Bind alias %s already in scope.", b.Variable()))
		}

		expr, err := this.Map(b.Expression())
		if err != nil {
			return err
		}

		b.SetExpression(expr)
		this.Allowed.SetField(b.Variable(), b.Variable())
	}

	return nil
}

func (this *Formalizer) PopBindings() {
	this.Allowed = this.Allowed.(*value.ScopeValue).Value
}

// Subquery

func (this *Formalizer) VisitSubquery(expr *Subquery) (interface{}, error) {
	err := expr.Select().FormalizeSubquery(this)
	if err != nil {
		return nil, err
	} else {
		return expr, nil
	}
}
