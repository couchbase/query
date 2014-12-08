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
	"github.com/couchbaselabs/query/value"
)

/*
An identifier is a symbolic reference to a particular value
in the current context. Type identifier is a struct that
implements ExpressionBase. It contains a variable identifier
of type string that represents identifiers.
*/
type Identifier struct {
	ExpressionBase
	identifier string
}

/*
This method returns a pointer to an Identifier structure
that has its identifier field populated by the input argument.
*/
func NewIdentifier(identifier string) Path {
	rv := &Identifier{
		identifier: identifier,
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitIdentifier method by passing in the receiver to
process identifier expressions, and returns the interface. It is
a visitor pattern.
*/
func (this *Identifier) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIdentifier(this)
}

/*
It returns JSON value that is all-encompassing.
*/
func (this *Identifier) Type() value.Type { return value.JSON }

/*
Call the Field method using the item value input argument on the
receiver. This returns a value. To evaluate an identifier, look
into the current item, find a field whose name is the
identifier, and return the value of that field within the current
item.
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

/*
Return the identifier string field of the receiver.
*/
func (this *Identifier) Alias() string {
	return this.identifier
}

/*
An identifier can be used as an index. Hence return true.
*/
func (this *Identifier) Indexable() bool {
	return true
}

/*
This method checks if the input expression is an Identifier type.
It it is return true if the identifiers are equal. If not return
false.
*/
func (this *Identifier) EquivalentTo(other Expression) bool {
	switch other := other.(type) {
	case *Identifier:
		return this.identifier == other.identifier
	default:
		return false
	}
}

/*
Since identifiers dont have children this returns nil.
*/
func (this *Identifier) Children() Expressions {
	return nil
}

/*
Returns nil.
*/
func (this *Identifier) MapChildren(mapper Mapper) error {
	return nil
}

func (this *Identifier) Copy() Expression {
	return this
}

/*
Call SetField using item value and set the identifier
string to the value. The SetField method returns a
boolean value. If it is nil return true since no error
was encountered while setting the field.
*/
func (this *Identifier) Set(item, val value.Value, context Context) bool {
	er := item.SetField(this.identifier, val)
	return er == nil
}

/*
Call UnsetFiled using item value and unset the identifier.
(delete it). The UnsetField returns a boolean value. If it
is nil return true since no error was encountered while
setting the field.
*/
func (this *Identifier) Unset(item value.Value, context Context) bool {
	er := item.UnsetField(this.identifier)
	return er == nil
}

/*
This method is used to access the identifier string
using the receiver.
*/
func (this *Identifier) Identifier() string {
	return this.identifier
}
