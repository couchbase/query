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
}

func NewField(first, second Expression) *Field {
	rv := &Field{
		BinaryFunctionBase: *NewBinaryFunctionBase("field", first, second),
	}

	switch second := second.(type) {
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

func (this *Field) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
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
		return (this.caseInsensitive == other.caseInsensitive) &&
			this.First().EquivalentTo(other.First()) &&
			this.Second().EquivalentTo(other.Second())
	default:
		return false
	}
}

func (this *Field) CoveredBy(keyspace string, exprs Expressions, options coveredOptions) Covered {
	for _, expr := range exprs {

		// MB-25560: if a field is equivalent, no need to check children field / field names
		if this.expr.EquivalentTo(expr) {
			return CoveredEquiv
		}
	}
	children := this.expr.Children()
	options.isSingle = len(children) == 1
	trickleEquiv := options.trickleEquiv
	options.trickleEquiv = true
	rv := CoveredTrue

	// MB-22112: we treat the special case where a keyspace is part of the projection list
	// a keyspace as a single term does not cover by definition
	// a keyspace as part of a field or a path does cover to delay the decision in terms
	// further down the path
	for _, child := range children {
		switch child.CoveredBy(keyspace, exprs, options) {
		case CoveredFalse:
			return CoveredFalse

		// MB-25317: ignore expressions not related to this keyspace
		case CoveredSkip:
			options.skip = true

		// MB-25650: this subexpression is already covered, no need to check subsequent terms
		case CoveredEquiv:
			options.skip = true

			// trickle down CoveredEquiv to outermost field
			if trickleEquiv {
				rv = CoveredEquiv
			}
		}
	}

	return rv
}

/*
Perform either case-sensitive or case-insensitive field lookup.
*/
func (this *Field) Apply(context Context, first, second value.Value) (value.Value, error) {
	switch second.Type() {
	case value.STRING:
		s := second.Actual().(string)
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
		if first.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else {
			return value.NULL_VALUE, nil
		}
	}
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
		er := first.SetField(second.Actual().(string), val)
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
		er := first.UnsetField(second.Actual().(string))
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
		return (this.name == other.name) &&
			(this.caseInsensitive == other.caseInsensitive)
	default:
		return this.valueEquivalentTo(other)
	}
}

// MB-22112 We need an ad hoc CoveredBy for FieldNames, so that we can make sure
// that they can be checked for equivalence against their natural match, identifiers
func (this *FieldName) CoveredBy(keyspace string, exprs Expressions, options coveredOptions) Covered {

	// MB-25317 / MB-25370 if the identifier preceeding the field name is not the keyspace
	// then we are skipping this test
	if options.skip {
		return CoveredSkip
	}
	for _, expr := range exprs {
		var isEquivalent bool

		switch eType := expr.(type) {
		case *FieldName:
			isEquivalent = (this.name == eType.name) &&
				(this.caseInsensitive == eType.caseInsensitive)
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
