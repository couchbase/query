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
	"github.com/couchbase/query/value"
)

/*
Identifier flags
*/
const (
	IDENT_IS_UNKNOWN      = 1 << iota // unknown
	IDENT_IS_KEYSPACE                 // keyspace or its alias or equivalent (e.g. subquery term)
	IDENT_IS_VARIABLE                 // binding variable
	IDENT_IS_PROJ_ALIAS               // alias used in projection
	IDENT_IS_UNNEST_ALIAS             // UNNEST alias
)

/*
An identifier is a symbolic reference to a particular value
in the current context.
*/
type Identifier struct {
	ExpressionBase
	identifier      string
	caseInsensitive bool
	parenthesis     bool
	identFlags      uint32
}

func NewIdentifier(identifier string) *Identifier {
	rv := &Identifier{
		identifier: identifier,
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Identifier) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIdentifier(this)
}

func (this *Identifier) Type() value.Type { return value.JSON }

/*
Evaluate this as a top-level identifier.
*/
func (this *Identifier) Evaluate(item value.Value, context Context) (value.Value, error) {
	rv, _ := item.Field(this.identifier)
	return rv, nil
}

/*
Value() returns the static / constant value of this Expression, or
nil. Expressions that depend on data, clocks, or random numbers must
return nil.
*/
func (this *Identifier) Value() value.Value {
	return nil
}

func (this *Identifier) Static() Expression {
	return nil
}

func (this *Identifier) Alias() string {
	return this.identifier
}

/*
An identifier can be used as an index. Hence return true.
*/
func (this *Identifier) Indexable() bool {
	return true
}

func (this *Identifier) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *Identifier:
		return (this.identifier == other.identifier) &&
			(this.caseInsensitive == other.caseInsensitive)
	default:
		return false
	}
}

func (this *Identifier) CoveredBy(keyspace string, exprs Expressions, options coveredOptions) Covered {
	// MB-25317, if this is not the right keyspace, ignore the expression altogether
	// MB-25370 this only applies for keyspace terms, not variables!
	if (this.IsKeyspaceAlias() && this.identifier != keyspace) ||
		this.IsProjectionAlias() || (!options.hasCoverBindVar() && this.IsBindingVariable()) {
		return CoveredSkip
	}

	for _, expr := range exprs {
		if this.EquivalentTo(expr) {
			switch eType := expr.(type) {
			case *Identifier:
				if options.hasCoverBindVar() && this.IsBindingVariable() {
					if eType.identifier == keyspace {
						return CoveredTrue
					} else {
						return CoveredSkip
					}
				} else {
					if !this.IsKeyspaceAlias() {
						return CoveredTrue
					} else if eType.identifier != keyspace {
						return CoveredSkip
					}
				}
			default:
				return CoveredTrue
			}
		}
	}

	return CoveredFalse
}

func (this *Identifier) Children() Expressions {
	return nil
}

func (this *Identifier) MapChildren(mapper Mapper) error {
	return nil
}

func (this *Identifier) Copy() Expression {
	return this
}

func (this *Identifier) SurvivesGrouping(groupKeys Expressions, allowed *value.ScopeValue) (
	bool, Expression) {
	for _, key := range groupKeys {
		if this.EquivalentTo(key) {
			return true, nil
		}
	}

	flags, found := allowed.Field(this.identifier)
	if found {
		allow_flags := uint32(flags.ActualForIndex().(int64))
		if (allow_flags & IDENT_IS_PROJ_ALIAS) != 0 {
			this.SetProjectionAlias(true)
		} else if (allow_flags & IDENT_IS_VARIABLE) != 0 {
			this.SetBindingVariable(true)
		}
		return true, nil
	}

	return false, this
}

func (this *Identifier) Set(item, val value.Value, context Context) bool {
	er := item.SetField(this.identifier, val)
	return er == nil
}

func (this *Identifier) Unset(item value.Value, context Context) bool {
	er := item.UnsetField(this.identifier)
	return er == nil
}

func (this *Identifier) Identifier() string {
	return this.identifier
}

func (this *Identifier) CaseInsensitive() bool {
	return this.caseInsensitive
}

func (this *Identifier) SetCaseInsensitive(insensitive bool) {
	this.caseInsensitive = insensitive
}

func (this *Identifier) Parenthesis() bool {
	return this.parenthesis
}

func (this *Identifier) SetParenthesis(parenthesis bool) {
	this.parenthesis = parenthesis
}

func (this *Identifier) IsKeyspaceAlias() bool {
	return (this.identFlags & IDENT_IS_KEYSPACE) != 0
}

func (this *Identifier) SetKeyspaceAlias(keyspaceAlias bool) {
	if keyspaceAlias {
		this.identFlags |= IDENT_IS_KEYSPACE
	} else {
		this.identFlags &^= IDENT_IS_KEYSPACE
	}
}

func (this *Identifier) IsBindingVariable() bool {
	return (this.identFlags & IDENT_IS_VARIABLE) != 0
}

func (this *Identifier) SetBindingVariable(bindingVariable bool) {
	if bindingVariable {
		this.identFlags |= IDENT_IS_VARIABLE
	} else {
		this.identFlags &^= IDENT_IS_VARIABLE
	}
}

func (this *Identifier) IsProjectionAlias() bool {
	return (this.identFlags & IDENT_IS_PROJ_ALIAS) != 0
}

func (this *Identifier) SetProjectionAlias(projectionAlias bool) {
	if projectionAlias {
		this.identFlags |= IDENT_IS_PROJ_ALIAS
	} else {
		this.identFlags &^= IDENT_IS_PROJ_ALIAS
	}
}

func (this *Identifier) IsUnnestAlias() bool {
	return (this.identFlags & IDENT_IS_UNNEST_ALIAS) != 0
}

func (this *Identifier) SetUnnestAlias(unnestAlias bool) {
	if unnestAlias {
		this.identFlags |= IDENT_IS_UNNEST_ALIAS
	} else {
		this.identFlags &^= IDENT_IS_UNNEST_ALIAS
	}
}

func (this *Identifier) SetIdentFlags(aliases map[string]bool, flags uint32) {
	if aliases != nil {
		if _, ok := aliases[this.identifier]; ok {
			this.identFlags |= flags
		}
	}
}
