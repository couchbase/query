//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"fmt"

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

func (this *Formalizer) Map(expr Expression) (Expression, error) {
	exp, err := expr.Accept(this)
	if err != nil {
		return nil, err
	}

	return exp.(Expression), nil
}

func (this *Formalizer) MapBindings() bool { return false }

// Arithmetic

func (this *Formalizer) VisitAdd(expr *Add) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitDiv(expr *Div) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitMod(expr *Mod) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitMult(expr *Mult) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitNeg(expr *Neg) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitSub(expr *Sub) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Case

func (this *Formalizer) VisitSearchedCase(expr *SearchedCase) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitSimpleCase(expr *SimpleCase) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Collection

func (this *Formalizer) VisitExists(expr *Exists) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitIn(expr *In) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitWithin(expr *Within) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitAny(expr *Any) (interface{}, error) {
	sv, err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(sv)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitEvery(expr *Every) (interface{}, error) {
	sv, err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(sv)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitArray(expr *Array) (interface{}, error) {
	sv, err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(sv)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitFirst(expr *First) (interface{}, error) {
	sv, err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(sv)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

// Comparison

func (this *Formalizer) VisitBetween(expr *Between) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitEq(expr *Eq) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitLE(expr *LE) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitLike(expr *Like) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitLT(expr *LT) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitIsMissing(expr *IsMissing) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitIsNull(expr *IsNull) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitIsValued(expr *IsValued) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Concat

func (this *Formalizer) VisitConcat(expr *Concat) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Constant

func (this *Formalizer) VisitConstant(expr *Constant) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Identifier

func (this *Formalizer) VisitIdentifier(expr *Identifier) (interface{}, error) {
	_, ok := this.Allowed.Field(expr.Identifier())
	if ok {
		return expr, nil
	}

	if this.Keyspace == "" {
		return nil, fmt.Errorf("Ambiguous reference to field %v.", expr.Identifier())
	}

	return NewField(
			NewIdentifier(this.Keyspace),
			NewFieldName(expr.Identifier())),
		nil
}

// Construction

func (this *Formalizer) VisitArrayConstruct(expr *ArrayConstruct) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitObjectConstruct(expr *ObjectConstruct) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Logic

func (this *Formalizer) VisitAnd(expr *And) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitNot(expr *Not) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitOr(expr *Or) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Navigation

func (this *Formalizer) VisitElement(expr *Element) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitField(expr *Field) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitFieldName(expr *FieldName) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

func (this *Formalizer) VisitSlice(expr *Slice) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Function

func (this *Formalizer) VisitFunction(expr Function) (interface{}, error) {
	return expr, expr.MapChildren(this)
}

// Subquery
func (this *Formalizer) VisitSubquery(expr Subquery) (interface{}, error) {
	err := expr.Formalize(this)
	if err != nil {
		return nil, err
	} else {
		return expr, nil
	}
}

// Parameters

func (this *Formalizer) VisitNamedParameter(expr NamedParameter) (interface{}, error) {
	return expr, nil
}

func (this *Formalizer) VisitPositionalParameter(expr PositionalParameter) (interface{}, error) {
	return expr, nil
}

// Bindings
func (this *Formalizer) PushBindings(bindings Bindings) (sv *value.ScopeValue, err error) {
	sv = value.NewScopeValue(make(map[string]interface{}, len(bindings)), this.Allowed)

	var expr Expression
	for _, b := range bindings {
		_, ok := this.Allowed.Field(b.Variable())
		if ok {
			return nil, fmt.Errorf("Bind alias %s already in scope.", b.Variable())
		}

		expr, err = this.Map(b.Expression())
		if err != nil {
			return nil, err
		}

		b.SetExpression(expr)
		sv.SetField(b.Variable(), b.Variable())
	}

	this.Allowed = sv
	return sv, nil
}

func (this *Formalizer) PopBindings(sv *value.ScopeValue) {
	this.Allowed = sv.Parent()
}
