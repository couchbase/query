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
	MapperBase

	Allowed  value.Value
	Keyspace string
}

func NewFormalizer() *Formalizer {
	rv := &Formalizer{
		Allowed: value.NewValue(make(map[string]interface{})),
	}

	rv.mapper = rv
	return rv
}

func (this *Formalizer) MapBindings() bool { return false }

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

// Subquery
func (this *Formalizer) VisitSubquery(expr Subquery) (interface{}, error) {
	err := expr.Formalize(this)
	if err != nil {
		return nil, err
	}

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
