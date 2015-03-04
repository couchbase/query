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

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Base64
//
///////////////////////////////////////////////////

/*
This represents the Meta function BASE64(expr). It returns
the base64-encoding of expr. Type Base64 is a struct that
implements UnaryFunctionBase.
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
	rv := &Base64{
		*NewUnaryFunctionBase("base64", operand),
	}

	rv.expr = rv
	return rv
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

	bytes, _ := operand.MarshalJSON() // Ignore errors from BINARY values
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
that implements UnaryFunctionBase.
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
	rv := &Meta{
		*NewUnaryFunctionBase("meta", operand),
	}

	rv.expr = rv
	return rv
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

var _SELF = NewSelf()

/*
The function NewSelf returns a pointer to the
NewNullaryFunctionBase to create a function SELF. It has
no input arguments.
*/
func NewSelf() Function {
	rv := &Self{
		*NewNullaryFunctionBase("self"),
	}

	rv.expr = rv
	return rv
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
	return func(operands ...Expression) Function { return _SELF }
}

///////////////////////////////////////////////////
//
// Uuid
//
///////////////////////////////////////////////////

/*
This represents the Meta function UUID(). It returns
a version 4 Universally Unique Identifier. Type Uuid
is a struct that implements NullaryFunctionBase.
*/
type Uuid struct {
	NullaryFunctionBase
}

/*
The function NewUuid returns a pointer to the NewNullaryFunctionBase
to create a function named UUID. It has no input arguments.
*/
func NewUuid() Function {
	rv := &Uuid{
		*NewNullaryFunctionBase("uuid"),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
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
Generate a Version 4 UUID as specified in RFC 4122, wrap it in a value
and return it. The UUID() function may return an error, if so return
a nil value UUID with the error.
*/
func (this *Uuid) Evaluate(item value.Value, context Context) (value.Value, error) {
	u, err := util.UUID()
	if err != nil {
		return nil, err
	}
	return value.NewValue(u), nil
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

///////////////////////////////////////////////////
//
// Version
//
///////////////////////////////////////////////////

/*
This represents the Meta function VERSION(). It returns
the current version of the N1QL server
*/
type Version struct {
	NullaryFunctionBase
}

/*
The function NewVersion returns a pointer to the NewNullaryFunctionBase
to create a function named Version. It has no input arguments.
*/
func NewVersion() Function {
	rv := &Version{
		*NewNullaryFunctionBase("version"),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Version) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a string value.
*/
func (this *Version) Type() value.Type { return value.STRING }

/*
Return the current server version, wrapped in a value.
*/
func (this *Version) Evaluate(item value.Value, context Context) (value.Value, error) {
	v := util.VERSION
	return value.NewValue(v), nil
}

/*
The constructor returns a NewVersion by casting the receiver to a
Function as the FunctionConstructor.
*/
func (this *Version) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return this
	}
}

///////////////////////////////////////////////////
//
// MinVersion
//
///////////////////////////////////////////////////

/*
This represents the Meta function MIN_VERSION(). It returns
the current minimum supported version of the N1QL server
*/
type MinVersion struct {
	NullaryFunctionBase
}

/*
The function NewMinVersion returns a pointer to the NewNullaryFunctionBase
to create a function named MinVersion. It has no input arguments.
*/
func NewMinVersion() Function {
	rv := &MinVersion{
		*NewNullaryFunctionBase("version"),
	}

	rv.volatile = true
	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *MinVersion) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a string value.
*/
func (this *MinVersion) Type() value.Type { return value.STRING }

/*
Return the current minimum version, wrapped in a value.
*/
func (this *MinVersion) Evaluate(item value.Value, context Context) (value.Value, error) {
	return value.NewValue(util.MIN_VERSION), nil
}

/*
The constructor returns a NewMinVersion by casting the receiver to a
Function as the FunctionConstructor.
*/
func (this *MinVersion) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return this
	}
}
