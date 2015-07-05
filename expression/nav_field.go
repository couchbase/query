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
are case sensitive. Type Field is a struct that implements
BinaryFunctionBase and has a boolean value caseInsensitive to
determine the case of the field name.
*/
type Field struct {
	BinaryFunctionBase
	caseInsensitive bool
}

/*
The function NewField calls NewBinaryFunctionBase to define the
field with input operand expressions first and second, as input.
*/
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
It calls the VisitField method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Field) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitField(this)
}

/*
It returns a value type JSON.
*/
func (this *Field) Type() value.Type { return value.JSON }

/*
Calls the Eval method for Binary functions and passes in the
receiver, current item and current context.
*/
func (this *Field) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
Return the expression alias of the receiver Field by
calling the Alias method on the second operand.
*/
func (this *Field) Alias() string {
	return this.Second().Alias()
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
The constructor returns a NewField with the operands
cast to a Function as the FunctionConstructor.
*/
func (this *Field) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewField(operands[0], operands[1])
	}
}

/*
Set Field to input value val. Evaluate the first and second operands
in the Field. If the first type is missing, set the target as the item.
If the second type is a string, call the SetField method to set the
string value second to the input value and return true if no error is
encountered during setting. For all other types return false.
*/
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

/*
Unset the Field value. Evaluate the first and second operands
in the Field. If the first type is missing, set the target as the item.
If the second type is a string, call the UnsetField method on the second
operand. Return true if no error is encountered during setting. For
all other types return false.
*/
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

/*
The function NewFieldName returns a pointer to a FieldName that
sets the name to the input expression.
*/
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
It calls the VisitFieldName method by passing in the receiver to
and returns the interface. It is a visitor pattern.
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
		return this.ValueEquals(other)
	}
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
