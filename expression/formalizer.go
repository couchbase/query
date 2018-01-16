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
Bit flags to indicate type of an identifier
*/
const (
	IDENT_IS_UNKNOWN  = 1 << iota // unknown
	IDENT_IS_KEYSPACE             // keyspace or its alias or equivalent (e.g. subquery term)
	IDENT_IS_VARIABLE             // binding variable
)

/*
Bit flags for formalizer flags
*/
const (
	FORM_MAP_SELF     = 1 << iota // Map SELF to keyspace: used in sarging index
	FORM_MAP_KEYSPACE             // Map keyspace to SELF: used in creating index
	FORM_IN_BINDING               // inside a binding scope
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
	flags       uint32
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
		mapSelf = mapSelf || parent.mapSelf()
		mapKeyspace = mapKeyspace || parent.mapKeyspace()
	}

	flags := uint32(0)
	if mapSelf {
		flags |= FORM_MAP_SELF
	}
	if mapKeyspace {
		flags |= FORM_MAP_KEYSPACE
	}

	rv := &Formalizer{
		keyspace:    keyspace,
		allowed:     value.NewScopeValue(make(map[string]interface{}), pv),
		identifiers: value.NewScopeValue(make(map[string]interface{}, 64), nil),
		aliases:     value.NewScopeValue(make(map[string]interface{}), av),
		flags:       flags,
	}

	if !mapKeyspace && keyspace != "" {
		rv.SetAllowedAlias(keyspace, true)
	}

	rv.mapper = rv
	return rv
}

func (this *Formalizer) mapSelf() bool {
	return (this.flags & FORM_MAP_SELF) != 0
}

func (this *Formalizer) mapKeyspace() bool {
	return (this.flags & FORM_MAP_KEYSPACE) != 0
}

func (this *Formalizer) inBinding() bool {
	return (this.flags & FORM_IN_BINDING) != 0
}

func (this *Formalizer) VisitAny(expr *Any) (interface{}, error) {
	inBinding, err := this.PushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(inBinding)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitEvery(expr *Every) (interface{}, error) {
	inBinding, err := this.PushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(inBinding)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitAnyEvery(expr *AnyEvery) (interface{}, error) {
	inBinding, err := this.PushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(inBinding)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitArray(expr *Array) (interface{}, error) {
	inBinding, err := this.PushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(inBinding)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitFirst(expr *First) (interface{}, error) {
	inBinding, err := this.PushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(inBinding)

	err = expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return expr, nil
}

func (this *Formalizer) VisitObject(expr *Object) (interface{}, error) {
	inBinding, err := this.PushBindings(expr.Bindings(), true)
	if err != nil {
		return nil, err
	}

	defer this.PopBindings(inBinding)

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

	ident_val, ok := this.allowed.Field(identifier)
	if ok {
		// if sarging for index, and not inside a binding scope,
		// then don't match with keyspace alias
		// (i.e., don't match an index key name with a keyspace alias)
		// however once we are in a binding scope, normal matching rules
		// apply, i.e., need to match with keyspace alias.
		ident_flags := uint32(ident_val.ActualForIndex().(int64))
		tmp_flags := ident_flags & IDENT_IS_KEYSPACE
		if !this.mapSelf() || this.inBinding() || tmp_flags == 0 {
			this.identifiers.SetField(identifier, ident_val)
			return expr, nil
		}
	}

	if this.keyspace == "" {
		return nil, fmt.Errorf("Ambiguous reference to field %v.", identifier)
	}

	if this.mapKeyspace() {
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
	if this.mapSelf() {
		return NewIdentifier(this.keyspace), nil
	} else {
		return expr, nil
	}
}

/*
Formalize META() functions defined on indexes.
*/
func (this *Formalizer) VisitFunction(expr Function) (interface{}, error) {
	if !this.mapKeyspace() {
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

func (this *Formalizer) PushBindings(bindings Bindings, push bool) (inBinding bool, err error) {
	inBinding = this.inBinding()

	allowed := this.allowed
	identifiers := this.identifiers
	aliases := this.aliases

	if push {
		allowed = value.NewScopeValue(make(map[string]interface{}, len(bindings)), this.allowed)
		identifiers = value.NewScopeValue(make(map[string]interface{}, 16), this.identifiers)
		aliases = value.NewScopeValue(make(map[string]interface{}, len(bindings)), this.aliases)
		this.flags |= FORM_IN_BINDING
	}

	var expr Expression
	var ident_flags uint32
	for _, b := range bindings {
		if ident_val, ok := allowed.Field(b.Variable()); ok {
			ident_flags = uint32(ident_val.ActualForIndex().(int64))
			tmp_flags1 := ident_flags & IDENT_IS_KEYSPACE
			tmp_flags2 := ident_flags &^ IDENT_IS_KEYSPACE
			// when sarging index keys, allow variables used in index definition
			// to be the same as a keyspace alias
			if !this.mapSelf() || tmp_flags1 == 0 || tmp_flags2 != 0 {
				err = fmt.Errorf("Duplicate variable %v already in scope.", b.Variable())
				return
			}
		} else {
			ident_flags = 0
		}

		ident_flags |= IDENT_IS_VARIABLE
		ident_val := value.NewValue(ident_flags)
		allowed.SetField(b.Variable(), ident_val)
		aliases.SetField(b.Variable(), ident_val)

		if b.NameVariable() != "" {
			if ident_val, ok := allowed.Field(b.NameVariable()); ok {
				ident_flags = uint32(ident_val.ActualForIndex().(int64))
				tmp_flags1 := ident_flags & IDENT_IS_KEYSPACE
				tmp_flags2 := ident_flags &^ IDENT_IS_KEYSPACE
				if !this.mapSelf() || tmp_flags1 == 0 || tmp_flags2 != 0 {
					err = fmt.Errorf("Duplicate variable %v already in scope.", b.NameVariable())
					return
				}
			} else {
				ident_flags = 0
			}

			ident_flags |= IDENT_IS_VARIABLE
			ident_val := value.NewValue(ident_flags)
			allowed.SetField(b.NameVariable(), ident_val)
			aliases.SetField(b.NameVariable(), ident_val)
		}

		expr, err = this.Map(b.Expression())
		if err != nil {
			return
		}

		b.SetExpression(expr)
	}

	if push {
		this.allowed = allowed
		this.identifiers = identifiers
		this.aliases = aliases
	}
	return
}

/*
Restore scope to parent's scope.
*/
func (this *Formalizer) PopBindings(inBinding bool) {

	currLevelAllowed := this.Allowed().GetValue().Fields()
	currLevelIndentfiers := this.Identifiers().GetValue().Fields()

	this.allowed = this.allowed.Parent().(*value.ScopeValue)
	this.identifiers = this.identifiers.Parent().(*value.ScopeValue)
	this.aliases = this.aliases.Parent().(*value.ScopeValue)

	if !inBinding {
		this.flags &^= FORM_IN_BINDING
	}

	// Identifiers that are used in current level but not defined in the current level scope move to parent
	for ident, _ := range currLevelIndentfiers {
		if currLevelAllowed != nil {
			if ident_val, ok := currLevelAllowed[ident]; !ok {
				this.identifiers.SetField(ident, ident_val)
			}
		}
	}
}

func (this *Formalizer) Copy() *Formalizer {
	f := NewFormalizer(this.keyspace, nil)
	f.allowed = this.allowed.Copy().(*value.ScopeValue)
	f.identifiers = this.identifiers.Copy().(*value.ScopeValue)
	f.aliases = this.aliases.Copy().(*value.ScopeValue)
	f.flags = this.flags
	return f
}

func (this *Formalizer) SetKeyspace(keyspace string) {
	this.keyspace = keyspace

	if !this.mapKeyspace() && keyspace != "" {
		this.SetAllowedAlias(keyspace, true)
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
		// we treat alias for keyspace as well as equivalent such as
		// subquery term, expression term, as same to keyspace
		var ident_flags uint32 = IDENT_IS_KEYSPACE
		this.aliases.SetField(alias, value.NewValue(ident_flags))
	}
}

// alias must be non-empty
func (this *Formalizer) SetAllowedAlias(alias string, isKeyspace bool) {
	var ident_flags uint32
	if isKeyspace {
		ident_flags = IDENT_IS_KEYSPACE
	} else {
		ident_flags = IDENT_IS_UNKNOWN
	}
	this.allowed.SetField(alias, value.NewValue(ident_flags))
}
