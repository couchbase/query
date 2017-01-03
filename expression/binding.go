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
	"encoding/json"
	"sort"
)

type Bindings []*Binding

/*
Binding is a helper class.
*/
type Binding struct {
	nameVariable string     `json:"name_var"`
	variable     string     `json:"var"`
	expr         Expression `json:"expr"`
	descend      bool       `json:"desc"`
}

func NewBinding(nameVariable, variable string, expr Expression, descend bool) *Binding {
	return &Binding{nameVariable, variable, expr, descend}
}

func NewSimpleBinding(variable string, expr Expression) *Binding {
	return &Binding{"", variable, expr, false}
}

func (this *Binding) Copy() *Binding {
	return &Binding{
		nameVariable: this.nameVariable,
		variable:     this.variable,
		expr:         this.expr.Copy(),
		descend:      this.descend,
	}
}

func (this *Binding) NameVariable() string {
	return this.nameVariable
}

func (this *Binding) Variable() string {
	return this.variable
}

func (this *Binding) Expression() Expression {
	return this.expr
}

func (this *Binding) SetExpression(expr Expression) {
	this.expr = expr
}

func (this *Binding) Descend() bool {
	return this.descend
}

func (this *Binding) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 4)
	if this.nameVariable != "" {
		r["name_var"] = this.nameVariable
	}
	r["var"] = this.variable
	r["expr"] = this.expr.String()
	if this.descend {
		r["desc"] = this.descend
	}

	return json.Marshal(r)
}

func (this Bindings) EquivalentTo(other Bindings) bool {
	if len(this) != len(other) {
		return false
	}

	for i, b := range this {
		o := other[i]
		if b.variable != o.variable ||
			b.descend != o.descend ||
			b.nameVariable != o.nameVariable ||
			!b.expr.EquivalentTo(o.expr) {
			return false
		}
	}

	return true
}

func (this Bindings) SubsetOf(other Bindings) bool {
	if len(this) != len(other) {
		return false
	}

	for i, b := range this {
		o := other[i]
		if b.variable != o.variable ||
			(b.descend && !o.descend) ||
			b.nameVariable != o.nameVariable ||
			!b.expr.EquivalentTo(o.expr) {
			return false
		}
	}

	return true
}

func (this Bindings) DependsOn(expr Expression) bool {
	for _, b := range this {
		if b.expr.DependsOn(expr) {
			return true
		}
	}

	return false
}

/*
Range over the bindings and map each expression to another.
*/
func (this Bindings) MapExpressions(mapper Mapper) (err error) {
	for _, b := range this {
		expr, err := mapper.Map(b.expr)
		if err != nil {
			return err
		}

		b.expr = expr
	}

	return
}

/*
   Returns all contained Expressions.
*/
func (this Bindings) Expressions() Expressions {
	exprs := make(Expressions, len(this))

	for i, b := range this {
		exprs[i] = b.expr
	}

	return exprs
}

func (this Bindings) Identifiers() Expressions {
	exprs := make(Expressions, 0, 2*len(this))

	for _, b := range this {
		if b.nameVariable != "" {
			exprs = append(exprs, NewIdentifier(b.nameVariable))
		}

		exprs = append(exprs, NewIdentifier(b.variable))
	}

	return exprs
}

func (this Bindings) Mappings() map[string]Expression {
	mappings := make(map[string]Expression, len(this))

	for _, b := range this {
		mappings[b.variable] = b.expr
	}

	return mappings
}

func (this Bindings) Copy() Bindings {
	copies := make(Bindings, len(this))
	for i, b := range this {
		copies[i] = b.Copy()
	}

	return copies
}

// Implement sort.Interface

func (this Bindings) Len() int {
	return len(this)
}

func (this Bindings) Less(i, j int) bool {
	return this[i].nameVariable < this[j].nameVariable ||
		(this[i].nameVariable == this[j].nameVariable &&
			this[i].variable < this[j].variable)
}

func (this Bindings) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this Bindings) Sort() {
	sort.Sort(this)
}
