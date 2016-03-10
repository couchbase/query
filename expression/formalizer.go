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

	"github.com/couchbase/query/value"
)

/*
Convert expressions to its full equivalent form.
Type Formalizer inherits from MapperBase. It has fields
Allowed and keyspace of type value and string.
*/
type Formalizer struct {
	MapperBase

	keyspace    string
	allowed     *value.ScopeValue
	identifiers map[string]bool
}

/*
This method returns a pointer to a Formalizer struct
with allowed set to a new map of type string to interface.
*/
func NewFormalizer(keyspace string, parent *Formalizer) *Formalizer {
	var pv value.Value
	if parent != nil {
		pv = parent.allowed
	}

	rv := &Formalizer{
		keyspace:    keyspace,
		allowed:     value.NewScopeValue(make(map[string]interface{}), pv),
		identifiers: make(map[string]bool),
	}

	if keyspace != "" {
		rv.allowed.SetField(keyspace, keyspace)
	}

	rv.mapper = rv
	return rv
}

/*
Visitor method for an Any Range Predicate that maps the
children of the input ANY expression.
*/
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

/*
Visitor method for an Every Range Predicate that maps the
children of the input EVERY expression.
*/
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

/*
Visitor method for an Any and Every Range Predicate that maps the
children of the input ANY AND EVERY expression.
*/
func (this *Formalizer) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
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

/*
Visitor method for an Array Range Transform that maps the
children of the input ARRAY expression.
*/
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

/*
Visitor method for an First Range Transform that maps the
children of the input FIRST expression.
*/
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

/*
Formalize Identifier.
*/
func (this *Formalizer) VisitIdentifier(expr *Identifier) (interface{}, error) {
	_, ok := this.allowed.Field(expr.Identifier())
	if ok {
		if this.identifiers != nil {
			this.identifiers[expr.Identifier()] = true
		}

		return expr, nil
	}

	if this.keyspace == "" {
		return nil, fmt.Errorf("Ambiguous reference to field %v.", expr.Identifier())
	}

	return NewField(
			NewIdentifier(this.keyspace),
			NewFieldName(expr.Identifier(), expr.CaseInsensitive())),
		nil
}

/*
Formalize META() functions defined on indexes.
*/
func (this *Formalizer) VisitFunction(expr Function) (interface{}, error) {
	meta, ok := expr.(*Meta)
	if ok && len(meta.Operands()) == 0 && this.keyspace != "" {
		return NewMeta(NewIdentifier(this.keyspace)), nil
	}

	return expr, expr.MapChildren(this.mapper)
}

/*
Visitor method for Subqueries. Call formalize.
*/
func (this *Formalizer) VisitSubquery(expr Subquery) (interface{}, error) {
	err := expr.Formalize(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

/*
Visitor method for Bindings. Value is a new map from string
to interface which is populated using the bindings in the
scope of the parent which is listed by the value allowed.
Bring the bindings that have parrent allowed into scope.
*/
func (this *Formalizer) PushBindings(bindings Bindings) (sv *value.ScopeValue, err error) {
	sv = value.NewScopeValue(make(map[string]interface{}, len(bindings)), this.allowed)

	var expr Expression
	for _, b := range bindings {
		_, ok := this.allowed.Field(b.Variable())
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

	this.allowed = sv
	return sv, nil
}

/*
Set scope to parent's scope.
*/
func (this *Formalizer) PopBindings(sv *value.ScopeValue) {
	parent := sv.Parent()
	if parent == nil {
		this.allowed = nil
	}

	this.allowed = sv.Parent().(*value.ScopeValue)
}

func (this *Formalizer) Copy() *Formalizer {
	f := NewFormalizer(this.keyspace, nil)
	f.allowed = this.allowed.Copy().(*value.ScopeValue)

	for ident, val := range this.identifiers {
		f.identifiers[ident] = val
	}

	return f
}

func (this *Formalizer) SetKeyspace(keyspace string) {
	this.keyspace = keyspace

	if keyspace != "" {
		this.allowed.SetField(keyspace, keyspace)
	}
}

func (this *Formalizer) Keyspace() string {
	return this.keyspace
}

func (this *Formalizer) Allowed() *value.ScopeValue {
	return this.allowed
}

func (this *Formalizer) SetIdentifiers(identifiers map[string]bool) {
	this.identifiers = identifiers
}

func (this *Formalizer) Identifiers() map[string]bool {
	return this.identifiers
}
