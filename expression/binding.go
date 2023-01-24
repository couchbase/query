//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	static       bool       `json:"static"`
}

const (
	BINDING_VARS_SAME = BindingVarOptions(iota)
	BINDING_VARS_CONFLICT
	BINDING_VARS_DIFFER
)

type BindingVarOptions uint32

func NewBinding(nameVariable, variable string, expr Expression, descend bool) *Binding {
	return &Binding{nameVariable, variable, expr, descend, false}
}

func NewSimpleBinding(variable string, expr Expression) *Binding {
	return &Binding{"", variable, expr, false, false}
}

func (this *Binding) Copy() *Binding {
	return &Binding{
		nameVariable: this.nameVariable,
		variable:     this.variable,
		expr:         this.expr.Copy(),
		descend:      this.descend,
		static:       this.static,
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

func (this *Binding) Static() bool {
	return this.static
}

func (this *Binding) SetStatic(s bool) {
	this.static = s
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
	if this.static {
		r["static"] = this.static
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

// this function allows two bindings to be considered equivalent if the binding variable
// names are different but everything else is the same; it then replaces binding variable
// names in the second expression with that of the first expression, then compare the two
// expressions to see whether they are equivalent
func equivalentBindingsWithExpression(bindings1, bindings2 Bindings, exprs1, exprs2 Expressions) bool {
	if len(bindings1) != len(bindings2) || len(exprs1) != len(exprs2) {
		return false
	}

	differ := false
	for i := 0; i < len(bindings1); i++ {
		b1 := bindings1[i]
		b2 := bindings2[i]
		if b1.descend != b2.descend ||
			!b1.expr.EquivalentTo(b2.expr) {
			return false
		}
		if !differ && (b1.variable != b2.variable || b1.nameVariable != b2.nameVariable) {
			differ = true
		}
	}

	renamer := NewRenamer(bindings2, bindings1)

	for i := 0; i < len(exprs2); i++ {
		expr1 := exprs1[i]
		expr2 := exprs2[i]
		if expr1 == nil && expr2 == nil {
			continue
		} else if expr1 == nil || expr2 == nil {
			return false
		}

		newExpr2 := expr2
		if differ {
			var err error
			newExpr2, err = renamer.Map(expr2.Copy())
			if err != nil {
				return false
			}
		}

		var equivalent bool
		switch exp1 := expr1.(type) {
		case CollPredicate:
			equivalent = exp1.EquivalentCollPred(newExpr2)
		case collMap:
			equivalent = exp1.EquivalentCollMap(newExpr2)
		default:
			equivalent = exp1.EquivalentTo(newExpr2)
		}
		if !equivalent {
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
		if (b.descend && !o.descend) ||
			(b.nameVariable != "" && o.nameVariable == "") ||
			!b.expr.EquivalentTo(o.expr) {
			return false
		}
	}

	return true
}

func (this Bindings) RenameVariables(other Bindings) (BindingVarOptions, map[string]bool) {
	if !this.SubsetOf(other) {
		return BINDING_VARS_SAME, nil
	}
	bnames := make(map[string]bool, 2*len(this))
	onames := make(map[string]bool, 2*len(this))
	for i, b := range this {
		o := other[i]
		if b.variable != o.variable {
			bnames[b.variable] = true
			onames[o.variable] = true
		}
		if b.nameVariable != o.nameVariable {
			if b.nameVariable != "" {
				bnames[b.nameVariable] = true
			}
			if o.nameVariable != "" {
				onames[o.nameVariable] = true
			}
		}
	}
	for n, _ := range onames {
		if _, ok := bnames[n]; ok {
			return BINDING_VARS_CONFLICT, onames
		}
	}

	return BINDING_VARS_DIFFER, onames
}

func (this Bindings) DuplicateVariable(names map[string]bool) bool {
	for _, b := range this {
		if _, ok := names[b.variable]; ok {
			return true
		}
		if b.nameVariable != "" {
			if _, ok := names[b.nameVariable]; ok {
				return true
			}
		}
	}
	return false
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
		var id *Identifier

		if b.nameVariable != "" {
			id = NewIdentifier(b.nameVariable)
			id.SetBindingVariable(true)
			if b.static {
				id.SetStaticVariable(true)
			}
			exprs = append(exprs, id)
		}

		id = NewIdentifier(b.variable)
		id.SetBindingVariable(true)
		if b.static {
			id.SetStaticVariable(true)
		}
		exprs = append(exprs, id)
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
