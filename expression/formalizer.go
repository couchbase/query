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

/*
Convert expressions to its full equivalent form. 
Type Formalizer inherits from MapperBase. It has fields
Allowed and keyspace of type value and string.
*/ 
type Formalizer struct {
	MapperBase

	Allowed  value.Value
	Keyspace string
}

/*
This method returns a pointer to a Formalizer struct
with Allowed set to a new map of type string to interface.
*/
func NewFormalizer() *Formalizer {
	rv := &Formalizer{
		Allowed: value.NewValue(make(map[string]interface{})),
	}

	rv.mapper = rv
	return rv
}

/*
Returns false for MapBindings.
*/
func (this *Formalizer) MapBindings() bool { return false }

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
Visitor method for an Identifier expressions. Check if the
expression Identifier is a field (in an object). If it is
return the expression. If the Keyspace string is empty 
for the receiver, there is an ambiguous reference to 
the field identifier. Hence throw an error. Return a new
Field with an identifier with the name Keyspace and Field
name set to the Identifier() return value.
*/
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
scope of the parent which is listed by the value Allowed.
Bring the bindings that have parrent Allowed into scope.
*/
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

/*
Set scope as parents scope.
*/
func (this *Formalizer) PopBindings(sv *value.ScopeValue) {
	this.Allowed = sv.Parent()
}
