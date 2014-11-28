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
	"encoding/base64"

	"github.com/couchbaselabs/query/value"

	"github.com/twinj/uuid"
)

///////////////////////////////////////////////////
//
// Base64
//
///////////////////////////////////////////////////

/*
This represents the Meta function BASE64(expr). It returns
the base64-encoding of expr. Type Base64 is a struct that
implements UnaryFuncitonBase.
*/
type Base64 struct {
	UnaryFunctionBase
}

/*
The function NewBase64 takes as input an expression and returns
a pointer to the Base64 struct that calls NewUnaryFunctionBase to
create a function named BASE64 with an input operand as the
expression.
*/
func NewBase64(operand Expression) Function {
	return &Base64{
		*NewUnaryFunctionBase("base64", operand),
	}
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Base64) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a String value.
*/
func (this *Base64) Type() value.Type { return value.STRING }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Base64) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns a value.
If the type of operand is missing then return it. Call MarshalJSON
to get the bytes, and then use Go's encoding/base64 package to
encode the bytes to string. Create a newValue using the string and
return it.
*/
func (this *Base64) Apply(context Context, operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return operand, nil
	}

	bytes, _ := operand.MarshalJSON()
	str := base64.StdEncoding.EncodeToString(bytes)
	return value.NewValue(str), nil
}

/*
The constructor returns a NewBase64 with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Base64) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBase64(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Meta
//
///////////////////////////////////////////////////

/*
This represents the Meta function META(expr). It returns
the meta data for the document expr. Type Meta is a struct
that implements UnaryFuncitonBase.
*/
type Meta struct {
	UnaryFunctionBase
}

/*
The function NewMeta takes as input an expression and returns
a pointer to the Meta struct that calls NewUnaryFunctionBase to
create a function named META with an input operand as the
expression.
*/
func NewMeta(operand Expression) Function {
	return &Meta{
		*NewUnaryFunctionBase("meta", operand),
	}
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Meta) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a OBJECT value.
*/
func (this *Meta) Type() value.Type { return value.OBJECT }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *Meta) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an operand value and context and returns a value.
If the type of operand is missing then return it. If the operand
type is AnnotatedValue then we call NewValue using the GetAttachment
method on the operand with input string meta. In the event the there
is no attachment present, the default case is to return a NULL value.
*/
func (this *Meta) Apply(context Context, operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return operand, nil
	}

	switch operand := operand.(type) {
	case value.AnnotatedValue:
		return value.NewValue(operand.GetAttachment("meta")), nil
	default:
		return value.NULL_VALUE, nil
	}
}

/*
The constructor returns a NewMeta with an operand cast to a
Function as the FunctionConstructor.
*/
func (this *Meta) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMeta(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Self
//
///////////////////////////////////////////////////

/*
This represents the Meta function SELF(). It makes the
result into a valid json value after removing the object
mapper. It is a type struct that implements
NullaryFunctionBase.
*/
type Self struct {
	NullaryFunctionBase
}

/*
The function NewSelf returns a pointer to the
NewNullaryFunctionBase to create a function SELF. It has
no input arguments.
*/
func NewSelf() Function {
	return &Self{
		*NewNullaryFunctionBase("self"),
	}
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Self) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a JSON value.
*/
func (this *Self) Type() value.Type { return value.JSON }

/*
Returns the input item.
*/
func (this *Self) Evaluate(item value.Value, context Context) (value.Value, error) {
	return item, nil
}

/*
Return the receiver as FunctionConstructor.
*/
func (this *Self) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return this }
}

///////////////////////////////////////////////////
//
// Uuid
//
///////////////////////////////////////////////////

/*
This represents the Meta function UUID(). It returns
a version 4 Universally Unique Identifier. Type Uuid
is a struct that implements NullaryFuncitonBase.
*/
type Uuid struct {
	NullaryFunctionBase
}

/*
The init method is used to set the format of the uuid output.
The current set format is CleanHyphen Format = "%x-%x-%x-%x%x-%x".
*/
func init() {
	uuid.SwitchFormat(uuid.CleanHyphen)
}

/*
The function NewUuid returns a pointer to the NewNullaryFunctionBase
to create a function named UUID. It has no input arguments.
*/
func NewUuid() Function {
	return &Uuid{
		*NewNullaryFunctionBase("uuid"),
	}
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Uuid) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a string value.
*/
func (this *Uuid) Type() value.Type { return value.STRING }

/*
Generate a Version 4 UUID as specified in RFC 4122 using
package github/com/twinj/uuid, function NewV4. This returns
a string. Call newValue and return it.
*/
func (this *Uuid) Evaluate(item value.Value, context Context) (value.Value, error) {
	u := uuid.NewV4()
	return value.NewValue(u.String()), nil
}

/*
It is not indexable.
*/
func (this *Uuid) Indexable() bool {
	return false
}

func (this *Uuid) EquivalentTo(other Expression) bool {
	return false
}

/*
The constructor returns a NewUuid by casting the receiver to a
Function as the FunctionConstructor.
*/
func (this *Uuid) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return this
	}
}
