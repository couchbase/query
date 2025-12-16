//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"strings"

	"github.com/couchbase/query/value"
)

/*
Nested expressions are used to access fields inside of objects.
This is done using the dot operator. By default the field names
are case sensitive.
*/
type Field struct {
	BinaryFunctionBase
	caseInsensitive bool
	parenthesis     bool
	cache           Expression
}

func NewField(first, second Expression) *Field {
	rv := &Field{}
	rv.Init("field", first, second)

	switch second := second.(type) {
	case *FieldName:
		rv.caseInsensitive = second.CaseInsensitive()
	case *Identifier:
		rv.caseInsensitive = second.CaseInsensitive()
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Field) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitField(this)
}

func (this *Field) Type() value.Type { return value.JSON }

/*
Perform either case-sensitive or case-insensitive field lookup
*/
func (this *Field) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	// only walk if after []
	walk := false
	if se, ok := this.operands[0].(*Slice); ok {
		walk = se.IsFromEmptyBrackets()
	}

	switch second.Type() {
	case value.STRING:
		// if the argument is a field name we must use it as is and not parse it
		_, fieldName := this.operands[1].(*FieldName)
		if fieldName {
			return this.doEvaluate(context, first, second, walk)
		}
		exp := this.cache
		static := this.operands[1].StaticNoVariable() != nil
		// only consider cached value if operand is static
		if exp == nil || !static {
			exp = NewIdentifier(second.ToString())
			exp.(*Identifier).SetCaseInsensitive(this.CaseInsensitive())
			if static {
				this.cache = exp
			}
		}
		return exp.Evaluate(first, context)
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		if first.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else {
			return value.NULL_VALUE, nil
		}
	}
}

// needed as logic externally accessed directly
// only evaluates literal field names
func (this *Field) DoEvaluate(context Context, first, second value.Value) (value.Value, error) {
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}
	return this.doEvaluate(context, first, second, false)
}

func (this *Field) doEvaluate(context Context, first, second value.Value, walk bool) (value.Value, error) {
	if walk && first.Type() == value.ARRAY {
		return this.doArrayEvaluate(context, first, second)
	}
	switch second.Type() {
	case value.STRING:
		s := second.ToString()
		v, ok := first.Field(s)
		if !ok && this.caseInsensitive {
			s = strings.ToLower(s)
			fields := first.Fields()
			for f, val := range fields {
				if s == strings.ToLower(f) {
					return value.NewValue(val), nil
				}
			}
		}
		return v, nil
	case value.MISSING:
		return value.MISSING_VALUE, nil
	default:
		return value.NULL_VALUE, nil
	}
}

// for each element in the array, look for the field in question and return all results as an array
// do NOT processes nested arrays recursively
func (this *Field) doArrayEvaluate(context Context, first, second value.Value) (value.Value, error) {
	rv := make([]interface{}, 0, 16)
	fa, _ := first.Actual().([]interface{})
	for _, ff := range fa {
		fv := value.NewValue(ff)
		v, e := this.doEvaluate(context, fv, second, true)
		if e != nil {
			return nil, e
		} else if v.Type() == value.ARRAY {
			// flatten first level
			for _, e := range v.Actual().([]interface{}) {
				rv = append(rv, e)
			}
		} else if v.Type() != value.MISSING {
			rv = append(rv, v)
		}
	}
	if len(rv) == 0 {
		return value.MISSING_VALUE, nil
	}
	return value.NewValue(rv), nil
}

func (this *Field) Alias() string {
	return this.Second().Alias()
}

func (this *Field) Indexable() bool {
	if !this.BinaryFunctionBase.Indexable() {
		return false
	}

	// MB-16772, MB-15916, MB-21971. For META() expressions, only
	// id, cas, and expiration are indexable.
	if _, ok := this.First().(*Meta); !ok {
		return true
	}

	second := this.Second().Value()
	if second == nil {
		return false
	}

	sv, ok := second.Actual().(string)
	return ok && (sv == "id" || sv == "cas" || sv == "expiration" || sv == "xattrs")
}

func (this *Field) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *Field:
		return this.First().EquivalentTo(other.First()) && this.Second().EquivalentTo(other.Second())
	default:
		return false
	}
}

func (this *Field) CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered {
	var rv Covered
	for _, expr := range exprs {

		// MB-25560: if a field is equivalent, no need to check children field / field names
		if this.expr.EquivalentTo(expr) {
			return CoveredEquiv
		}

		// special handling of array index expression
		if options.hasCoverArrayKeyOptions() {
			if all, ok := expr.(*All); ok {
				rv = chkArrayKeyCover(this.expr, keyspace, exprs, all, options)
				if rv == CoveredTrue || rv == CoveredEquiv {
					return rv
				}
			}
		}
	}

	// no need to look at children if requesting binding var or binding expr
	// (requires exact match)
	if options.hasCoverBindExpr() || options.hasCoverBindVar() {
		return CoveredFalse
	}

	children := this.expr.Children()
	trickle := options.hasCoverTrickle()
	options.setCoverTrickle()
	rv = CoveredTrue

	// MB-22112: we treat the special case where a keyspace is part of the projection list
	// a keyspace as a single term does not cover by definition
	// a keyspace as part of a field or a path does cover to delay the decision in terms
	// further down the path
	for i, child := range children {
		switch child.CoveredBy(keyspace, exprs, options) {
		case CoveredFalse:
			return CoveredFalse

		// MB-25317: ignore expressions not related to this keyspace
		case CoveredSkip:
			options.setCoverSkip()

			// MB-30350 trickle down CoveredSkip to outermost field
			if trickle {
				rv = CoveredSkip
			}

		// MB-25560: this subexpression is already covered, no need to check subsequent terms
		case CoveredEquiv:
			options.setCoverSkip()

			// trickle down CoveredEquiv to outermost field
			if trickle {
				rv = CoveredEquiv
			}
		case CoveredTrue:
			if i == 0 {
				switch child.(type) {
				case NamedParameter, PositionalParameter:
					options.setCoverSkip()
				}
			}
		}
	}

	return rv
}

/*
Factory method pattern.
*/
func (this *Field) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewField(operands[0], operands[1])
	}
}

func (this *Field) Set(item, val value.Value, context Context) bool {
	first, er := this.First().Evaluate(item, context)
	if er != nil {
		return false
	}

	second, er := this.Second().Evaluate(item, context)
	if er != nil {
		return false
	}

	switch second.Type() {
	case value.STRING:
		er := first.SetField(second.ToString(), val)
		return er == nil
	default:
		return false
	}
}

func (this *Field) Unset(item value.Value, context Context) bool {
	second, er := this.Second().Evaluate(item, context)
	if er != nil {
		return false
	}

	first, er := this.First().Evaluate(item, context)
	if er != nil {
		return false
	}

	switch second.Type() {
	case value.STRING:
		er := first.UnsetField(second.ToString())
		return er == nil
	default:
		return false
	}
}

/*
Returns a boolean value that depicts if the Field is case
sensitive or not.
*/
func (this *Field) CaseInsensitive() bool {
	return this.caseInsensitive
}

/*
Set the fields case sensitivity to the input boolean value.
*/
func (this *Field) SetCaseInsensitive(insensitive bool) {
	this.caseInsensitive = insensitive
}

/*
Get the correct xattr path. For meta().xattr._sync, it gives _sync.
*/
func (this *Field) FieldNames(base Expression, names map[string]bool) (present bool) {
	if Equivalent(base, this.First()) {
		second := this.Second().Value()
		if second != nil {
			if sv, ok := second.Actual().(string); ok {
				names[sv] = this.caseInsensitive
			}
		}
		return true
	}

	for _, child := range this.Children() {
		if child.FieldNames(base, names) {
			present = true
		}
	}

	return present
}

func (this *Field) Parenthesis() bool {
	return this.parenthesis
}

func (this *Field) SetParenthesis(parenthesis bool) {
	this.parenthesis = parenthesis
}

func (this *Field) Path() []string {
	var out []string

outer:
	switch first := this.First().(type) {
	case *Field:
		switch one := first.First().(type) {
		case *Identifier:
			out = append(out, one.Alias())
		default:
			break outer
		}
		switch two := first.Second().(type) {
		case *FieldName:
			out = append(out, two.name)
		default:
			break outer
		}
		switch three := this.Second().(type) {
		case *FieldName:
			out = append(out, three.name)
		}
	case *Identifier:
		out = append(out, first.Alias())
		switch two := this.Second().(type) {
		case *FieldName:
			out = append(out, two.name)
		}
	}
	return out
}

func (this *Field) GetFirstCaseSensitivePathElement() Expression {
	switch first := this.First().(type) {
	case *Field:
		if i, ok := first.First().(*Identifier); ok && i.CaseInsensitive() {
			return i
		}
		if fn, ok := first.Second().(*FieldName); ok && fn.CaseInsensitive() {
			return fn
		}
	case *Identifier:
		if first.CaseInsensitive() {
			return first
		}
	}
	if fn, ok := this.Second().(*FieldName); ok && fn.CaseInsensitive() {
		return fn
	}
	if this.CaseInsensitive() {
		return this
	}
	return nil
}

func (this *Field) Copy() Expression {
	rv := this.BinaryFunctionBase.Copy().(*Field)
	rv.caseInsensitive = this.caseInsensitive
	return rv
}

func (this *Field) AssignTo(parent Expression) {
	switch t := this.First().(type) {
	case *Identifier:
		f := NewField(parent, NewFieldName(t.Alias(), t.caseInsensitive))
		this.operands[0] = f
	case *Field:
		t.AssignTo(parent)
	}
}

/*
FieldName represents the Field. It implements Constant and has a field
name as string. This class overrides the Alias() method so that the
field name is used as the alias.
*/
type FieldName struct {
	Constant
	name            string
	caseInsensitive bool
}

func NewFieldName(name string, caseInsensitive bool) Expression {
	rv := &FieldName{
		Constant: Constant{
			value: value.NewValue(name),
		},
		name:            name,
		caseInsensitive: caseInsensitive,
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *FieldName) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFieldName(this)
}

/*
Return the name of the Field as its Alias.
*/
func (this *FieldName) Alias() string {
	return this.name
}

func (this *FieldName) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *FieldName:
		return (this.name == other.name) ||
			((this.caseInsensitive || other.caseInsensitive) &&
				strings.ToLower(this.name) == strings.ToLower(other.name))
	default:
		return this.valueEquivalentTo(other)
	}
}

// MB-22112 We need an ad hoc CoveredBy for FieldNames, so that we can make sure
// that they can be checked for equivalence against their natural match, identifiers
func (this *FieldName) CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered {

	// MB-25317 / MB-25370 if the identifier preceeding the field name is not the keyspace
	// then we are skipping this test
	if options.hasCoverSkip() {
		return CoveredSkip
	}
	for _, expr := range exprs {
		var isEquivalent bool

		switch eType := expr.(type) {
		case *FieldName:
			isEquivalent = ((this.caseInsensitive || eType.caseInsensitive) &&
				strings.ToLower(this.name) == strings.ToLower(eType.name)) ||
				this.name == eType.name
		case *Identifier:
			isEquivalent = (this.caseInsensitive &&
				strings.ToLower(this.name) == strings.ToLower(eType.identifier)) ||
				this.name == eType.identifier
		default:
			isEquivalent = false
		}

		if isEquivalent {

			// MV-25560 if a field is covered, so are the sub elements
			return CoveredEquiv
		}
	}
	return CoveredFalse
}

/*
Constants are not transformed, so no need to copy.
*/
func (this *FieldName) Copy() Expression {
	return this
}

func (this *FieldName) CaseInsensitive() bool {
	return this.caseInsensitive
}
