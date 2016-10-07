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

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

/*
Convert expressions to full form qualified by keyspace aliases.
*/
type Formalizer struct {
	MapperBase

	keyspace    string
	allowed     *value.ScopeValue
	identifiers *value.ScopeValue
	aliases     *value.ScopeValue
	mapSelf     bool // Map SELF to keyspace: used in sarging index
	mapKeyspace bool // Map keyspace to SELF: used in creating index
}

func NewFormalizer(keyspace string, parent *Formalizer) *Formalizer {
	return newFormalizer(keyspace, parent, false, false)
}

func NewSelfFormalizer(keyspace string, parent *Formalizer) *Formalizer {
	return newFormalizer(keyspace, parent, true, false)
}

func NewKeyspaceFormalizer(keyspace string, parent *Formalizer) *Formalizer {
	return newFormalizer(keyspace, parent, false, true)
}

func newFormalizer(keyspace string, parent *Formalizer, mapSelf, mapKeyspace bool) *Formalizer {
	var pv, av value.Value
	if parent != nil {
		pv = parent.allowed
		av = parent.aliases
		mapSelf = mapSelf || parent.mapSelf
		mapKeyspace = mapKeyspace || parent.mapKeyspace
	}

	rv := &Formalizer{
		keyspace:    keyspace,
		allowed:     value.NewScopeValue(make(map[string]interface{}), pv),
		identifiers: value.NewScopeValue(make(map[string]interface{}, 64), nil),
		aliases:     value.NewScopeValue(make(map[string]interface{}), av),
		mapSelf:     mapSelf,
		mapKeyspace: mapKeyspace,
	}

	if !mapKeyspace && keyspace != "" {
		rv.allowed.SetField(keyspace, keyspace)
	}

	rv.mapper = rv
	return rv
}

func (this *Formalizer) VisitAny(expr *Any) (interface{}, error) {
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitEvery(expr *Every) (interface{}, error) {
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitArray(expr *Array) (interface{}, error) {
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitFirst(expr *First) (interface{}, error) {
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings()

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitObject(expr *Object) (interface{}, error) {
	err := this.PushBindings(expr.Bindings())
	if err != nil {
		return nil, err
	}

	defer this.PopBindings()

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
	identifier := expr.Identifier()

	_, ok := this.allowed.Field(identifier)
	if ok {
		this.identifiers.SetField(identifier, value.TRUE_VALUE)
		return expr, nil
	}

	if this.keyspace == "" {
		return nil, fmt.Errorf("Ambiguous reference to field %v.", identifier)
	}

	if this.mapKeyspace {
		if identifier == this.keyspace {
			return SELF, nil
		} else {
			return expr, nil
		}
	} else {
		return NewField(NewIdentifier(this.keyspace),
				NewFieldName(identifier, expr.CaseInsensitive())),
			nil
	}
}

/*
Formalize SELF functions defined on indexes.
*/
func (this *Formalizer) VisitSelf(expr *Self) (interface{}, error) {
	if this.mapSelf {
		return NewIdentifier(this.keyspace), nil
	} else {
		return expr, nil
	}
}

/*
Formalize META() functions defined on indexes.
*/
func (this *Formalizer) VisitFunction(expr Function) (interface{}, error) {
	if !this.mapKeyspace {
		meta, ok := expr.(*Meta)
		if ok && len(meta.Operands()) == 0 {
			if this.keyspace != "" {
				return NewMeta(NewIdentifier(this.keyspace)), nil
			} else {
				return nil, errors.NewAmbiguousMetaError()
			}
		}
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
Create new scope containing bindings.
*/
func (this *Formalizer) PushBindings(bindings Bindings) (err error) {
	allowed := value.NewScopeValue(make(map[string]interface{}, len(bindings)), this.allowed)
	identifiers := value.NewScopeValue(make(map[string]interface{}, 16), this.identifiers)
	aliases := value.NewScopeValue(make(map[string]interface{}, len(bindings)), this.aliases)

	var expr Expression
	for _, b := range bindings {
		expr, err = this.Map(b.Expression())
		if err != nil {
			return err
		}

		b.SetExpression(expr)
		allowed.SetField(b.Variable(), value.TRUE_VALUE)
		aliases.SetField(b.Variable(), value.TRUE_VALUE)
		if b.NameVariable() != "" {
			allowed.SetField(b.NameVariable(), value.TRUE_VALUE)
			aliases.SetField(b.NameVariable(), value.TRUE_VALUE)
		}
	}

	this.allowed = allowed
	this.identifiers = identifiers
	this.aliases = aliases
	return nil
}

/*
Restore scope to parent's scope.
*/
func (this *Formalizer) PopBindings() {
	this.allowed = this.allowed.Parent().(*value.ScopeValue)
	this.identifiers = this.identifiers.Parent().(*value.ScopeValue)
	this.aliases = this.aliases.Parent().(*value.ScopeValue)
}

func (this *Formalizer) Copy() *Formalizer {
	f := NewFormalizer(this.keyspace, nil)
	f.allowed = this.allowed.Copy().(*value.ScopeValue)
	f.identifiers = this.identifiers.Copy().(*value.ScopeValue)
	f.aliases = this.aliases.Copy().(*value.ScopeValue)
	f.mapSelf = this.mapSelf
	f.mapKeyspace = this.mapKeyspace
	return f
}

func (this *Formalizer) SetKeyspace(keyspace string) {
	this.keyspace = keyspace

	if !this.mapKeyspace && keyspace != "" {
		this.allowed.SetField(keyspace, keyspace)
	}
}

func (this *Formalizer) Keyspace() string {
	return this.keyspace
}

func (this *Formalizer) Allowed() *value.ScopeValue {
	return this.allowed
}

func (this *Formalizer) Identifiers() *value.ScopeValue {
	return this.identifiers
}

func (this *Formalizer) Aliases() *value.ScopeValue {
	return this.aliases
}

// Argument must be non-nil
func (this *Formalizer) SetIdentifiers(identifiers *value.ScopeValue) {
	this.identifiers = identifiers
}

func (this *Formalizer) SetAlias(alias string) {
	if alias != "" {
		this.aliases.SetField(alias, alias)
	}
}
