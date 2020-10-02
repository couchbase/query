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
// Base64Encode
//
///////////////////////////////////////////////////

/*
This represents the function BASE64_ENCODE(expr). It returns the
base64-encoding of expr.
*/
type Base64Encode struct {
	UnaryFunctionBase
}

func NewBase64Encode(operand Expression) Function {
	rv := &Base64Encode{
		*NewUnaryFunctionBase("base64_encode", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Base64Encode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Base64Encode) Type() value.Type { return value.STRING }

func (this *Base64Encode) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Base64Encode) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	bytes, _ := arg.MarshalJSON() // Ignore errors from BINARY values
	str := base64.StdEncoding.EncodeToString(bytes)
	return value.NewValue(str), nil
}

/*
Factory method pattern.
*/
func (this *Base64Encode) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBase64Encode(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Base64Decode
//
///////////////////////////////////////////////////

/*
This represents the function BASE64_DECODE(expr). It returns the
base64-decoding of expr.
*/
type Base64Decode struct {
	UnaryFunctionBase
}

func NewBase64Decode(operand Expression) Function {
	rv := &Base64Decode{
		*NewUnaryFunctionBase("base64_decode", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Base64Decode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Base64Decode) Type() value.Type { return value.STRING }

func (this *Base64Decode) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Base64Decode) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	str, err := base64.StdEncoding.DecodeString(arg.Actual().(string))
	if err != nil {
		return value.NULL_VALUE, nil
	} else {
		return value.NewValue(str), nil
	}
}

/*
Factory method pattern.
*/
func (this *Base64Decode) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewBase64Decode(operands[0])
	}
}

///////////////////////////////////////////////////
//
// Meta
//
///////////////////////////////////////////////////

/*
This represents the Meta function META(expr).
*/
type Meta struct {
	FunctionBase
}

func NewMeta(operands ...Expression) Function {
	rv := &Meta{
		*NewFunctionBase("meta", operands...),
	}

	if len(operands) > 0 {
		if ident, ok := operands[0].(*Identifier); ok {
			ident.SetKeyspaceAlias(true)
		}
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Meta) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Meta) Type() value.Type { return value.OBJECT }

func (this *Meta) Evaluate(item value.Value, context Context) (value.Value, error) {
	val := item

	if len(this.operands) > 0 {
		arg, err := this.operands[0].Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		val = arg
	}

	if val.Type() == value.MISSING {
		return val, nil
	}

	switch val := val.(type) {
	case value.AnnotatedValue:
		return value.NewValue(val.GetMeta()), nil
	default:
		return value.NULL_VALUE, nil
	}
}

func (this *Meta) Indexable() bool {
	return true
}

func (this *Meta) CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered {
	if len(this.operands) > 0 {
		alias := NewIdentifier(keyspace)
		if !this.operands[0].DependsOn(alias) {

			// MB-22561: skip the rest of the expression if different keyspace
			return CoveredSkip
		}
	}

	for _, expr := range exprs {
		if this.EquivalentTo(expr) {
			return CoveredTrue
		}
	}

	return CoveredFalse
}

func (this *Meta) MinArgs() int { return 0 }

func (this *Meta) MaxArgs() int { return 1 }

/*
Factory method pattern.
*/
func (this *Meta) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMeta(operands...)
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
mapper.
*/
type Self struct {
	NullaryFunctionBase
}

var SELF = NewSelf()

func NewSelf() Function {
	rv := &Self{
		*NewNullaryFunctionBase("self"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Self) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSelf(this)
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

func (this *Self) Indexable() bool {
	return true
}

func (this *Self) CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered {
	return CoveredFalse
}

func (this *Self) SurvivesGrouping(groupKeys Expressions, allowed *value.ScopeValue) (
	bool, Expression) {
	for _, key := range groupKeys {
		if this.EquivalentTo(key) {
			return true, nil
		}
	}

	return false, nil
}

/*
Factory method pattern.
*/
func (this *Self) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return SELF }
}

///////////////////////////////////////////////////
//
// Uuid
//
///////////////////////////////////////////////////

/*
This represents the Meta function UUID(). It returns
a version 4 Universally Unique Identifier.
*/
type Uuid struct {
	NullaryFunctionBase
}

func NewUuid() Function {
	rv := &Uuid{
		*NewNullaryFunctionBase("uuid"),
	}

	rv.setVolatile()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Uuid) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Uuid) Type() value.Type { return value.STRING }

/*
Generate a Version 4 UUID as specified in RFC 4122, wrap it in a value
and return it. The UUID() function may return an error, if so return
a nil value UUID with the error.
*/
func (this *Uuid) Evaluate(item value.Value, context Context) (value.Value, error) {
	u, err := util.UUIDV3()
	if err != nil {
		return nil, err
	}
	return value.NewValue(u), nil
}

/*
Factory method pattern.
*/
func (this *Uuid) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewUuid()
	}
}

///////////////////////////////////////////////////
//
// Version
//
///////////////////////////////////////////////////

/*
This represents the Meta function VERSION(). It returns
the current version of N1QL.
*/
type Version struct {
	NullaryFunctionBase
}

func NewVersion() Function {
	rv := &Version{
		*NewNullaryFunctionBase("version"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Version) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Version) Type() value.Type { return value.STRING }

/*
Return the current server version, wrapped in a value.
*/
func (this *Version) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _VERSION_VALUE, nil
}

var _VERSION_VALUE = value.NewValue(util.VERSION)

func (this *Version) Value() value.Value {
	return _VERSION_VALUE
}

func (this *Version) Indexable() bool {
	return false
}

/*
Factory method pattern.
*/
func (this *Version) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewVersion()
	}
}

///////////////////////////////////////////////////
//
// MinVersion
//
///////////////////////////////////////////////////

/*
This represents the function MIN_VERSION(). It returns
the current minimum supported version of N1QL.
*/
type MinVersion struct {
	NullaryFunctionBase
}

func NewMinVersion() Function {
	rv := &MinVersion{
		*NewNullaryFunctionBase("min_version"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *MinVersion) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *MinVersion) Type() value.Type { return value.STRING }

/*
Return the current minimum version, wrapped in a value.
*/
func (this *MinVersion) Evaluate(item value.Value, context Context) (value.Value, error) {
	return _MIN_VERSION_VALUE, nil
}

var _MIN_VERSION_VALUE = value.NewValue(util.MIN_VERSION)

func (this *MinVersion) Value() value.Value {
	return _MIN_VERSION_VALUE
}

func (this *MinVersion) Indexable() bool {
	return false
}

/*
Factory method pattern.
*/
func (this *MinVersion) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewMinVersion()
	}
}

///////////////////////////////////////////////////
//
// CurrentUsers
//
///////////////////////////////////////////////////

/*
This represents the array function CURRENT_USERS(). It
returns the authenticated users of the query as an array
of strings.
*/
type CurrentUsers struct {
	NullaryFunctionBase
}

func NewCurrentUsers() Function {
	rv := &CurrentUsers{
		*NewNullaryFunctionBase("current_users"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *CurrentUsers) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *CurrentUsers) Type() value.Type { return value.ARRAY }

func (this *CurrentUsers) Evaluate(item value.Value, context Context) (value.Value, error) {
	authUsers := context.AuthenticatedUsers()
	arr := make([]interface{}, len(authUsers))
	for i, user := range authUsers {
		arr[i] = user
	}
	arrVal := value.NewValue(arr)
	return arrVal, nil
}

func (this *CurrentUsers) Static() Expression {
	return this
}

/*
Factory method pattern.
*/
func (this *CurrentUsers) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function { return NewCurrentUsers() }
}

///////////////////////////////////////////////////
//
// DsVersion
//
///////////////////////////////////////////////////

/*
This represents the Meta function DS_VERSION(). It returns
the current version of the server, a string like "4.7.0-1544-enterprise".
*/
type DsVersion struct {
	NullaryFunctionBase
}

func NewDsVersion() Function {
	rv := &DsVersion{
		*NewNullaryFunctionBase("ds_version"),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *DsVersion) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DsVersion) Type() value.Type { return value.STRING }

/*
Return the current server version, wrapped in a value.
*/
func (this *DsVersion) Evaluate(item value.Value, context Context) (value.Value, error) {
	version := context.DatastoreVersion()
	return value.NewValue(version), nil
}

func (this *DsVersion) Indexable() bool {
	return false
}

/*
Factory method pattern.
*/
func (this *DsVersion) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDsVersion()
	}
}
