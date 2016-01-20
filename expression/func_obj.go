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
	"sort"

	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// ObjectLength
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_LENGTH(expr).
It returns the number of name-value pairs in the object.
Type ObjectLength is a struct that implements
UnaryFunctionBase.
*/
type ObjectLength struct {
	UnaryFunctionBase
}

/*
The function NewObjectLength calls NewUnaryFunctionBase to
create a function named OBJECT_LENGTH with an expression as
input.
*/
func NewObjectLength(operand Expression) Function {
	rv := &ObjectLength{
		*NewUnaryFunctionBase("object_length", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ObjectLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type NUMBER.
*/
func (this *ObjectLength) Type() value.Type { return value.NUMBER }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ObjectLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method returns the length of the object. If the type of
input is missing then return a missing value, and if not an
object return a null value. Convert it to a valid Go type.
Cast it to a map from string to interface and return its
length by using the len function by casting it to float64.
*/
func (this *ObjectLength) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	return value.NewValue(float64(len(oa))), nil
}

/*
The constructor returns a NewObjectLength with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *ObjectLength) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectLength(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectNames
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_NAMES(expr).
It returns an array containing the attribute names of
the object, in N1QL collation order. Type ObjectNames
is a struct that implements UnaryFunctionBase.
*/
type ObjectNames struct {
	UnaryFunctionBase
}

/*
The function NewObjectNames calls NewUnaryFunctionBase to
create a function named OBJECT_NAMES with an expression as
input.
*/
func NewObjectNames(operand Expression) Function {
	rv := &ObjectNames{
		*NewUnaryFunctionBase("object_names", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ObjectNames) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type ARRAY.
*/
func (this *ObjectNames) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ObjectNames) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an object and returns a slice of values
that contains the attribute names. If the type of input is
missing then return a missing value, and if not an
object return a null value. Convert it to a valid Go type.
Cast it to a map from string to interface. Range over this
map and retrieve the keys. Sort it and then use it to save
the corresponding values into a slice of interfaces. Return
the slice.
*/
func (this *ObjectNames) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	keys := make(sort.StringSlice, 0, len(oa))
	for key, _ := range oa {
		keys = append(keys, key)
	}

	sort.Sort(keys)
	ra := make([]interface{}, len(keys))
	for i, k := range keys {
		ra[i] = k
	}

	return value.NewValue(ra), nil
}

/*
The constructor returns a NewObjectNames with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *ObjectNames) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectNames(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectPairs
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_PAIRS(expr).
It returns an array containing the attribute name and
value pairs of the object, in N1QL collation order of
the names. Type ObjectPairs is a struct that implements
UnaryFunctionBase.
*/
type ObjectPairs struct {
	UnaryFunctionBase
}

/*
The function NewObjectPairs calls NewUnaryFunctionBase to
create a function named OBJECT_PAIRS with an expression as
input.
*/
func NewObjectPairs(operand Expression) Function {
	rv := &ObjectPairs{
		*NewUnaryFunctionBase("object_pairs", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ObjectPairs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type ARRAY.
*/
func (this *ObjectPairs) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ObjectPairs) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an object and returns a map of name
value pairs. If the type of input is missing then return
a missing value, and if not an object return a null value.
Convert it to a valid Go type. Cast it to a map from
string to interface. Range over this map and save the keys.
Sort the keys and range over the keys to create name and value
pairs. Return this object.
*/
func (this *ObjectPairs) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	keys := make(sort.StringSlice, 0, len(oa))
	for key, _ := range oa {
		keys = append(keys, key)
	}

	sort.Sort(keys)
	ra := make([]interface{}, len(keys))
	for i, k := range keys {
		ra[i] = map[string]interface{}{"name": k, "value": oa[k]}
	}

	return value.NewValue(ra), nil
}

/*
The constructor returns a NewObjectPairs with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *ObjectPairs) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectPairs(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectValues
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_VALUES(expr).
It returns an array containing the attribute values of
the object, in N1QL collation order of the corresponding
names. Type ObjectValues is a struct that implements
UnaryFunctionBase.
*/
type ObjectValues struct {
	UnaryFunctionBase
}

/*
The function NewObjectValues calls NewUnaryFunctionBase to
create a function named OBJECT_VALUES with an expression as
input.
*/
func NewObjectValues(operand Expression) Function {
	rv := &ObjectValues{
		*NewUnaryFunctionBase("object_values", operand),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ObjectValues) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type ARRAY.
*/
func (this *ObjectValues) Type() value.Type { return value.ARRAY }

/*
Calls the Eval method for unary functions and passes in the
receiver, current item and current context.
*/
func (this *ObjectValues) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

/*
This method takes in an object and returns a slice
that contains the attribute values. If the type of
input is missing then return a missing value, and
if not an object return a null value. Convert it to
a valid Go type. Cast it to a map from string to
interface. Range over this map and retrieve the keys.
Sort it and then use it to save the corresponding
values into a slice of interfaces. Return the slice.
*/
func (this *ObjectValues) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	keys := make(sort.StringSlice, 0, len(oa))
	for key, _ := range oa {
		keys = append(keys, key)
	}

	sort.Sort(keys)
	ra := make([]interface{}, len(keys))
	for i, k := range keys {
		ra[i] = oa[k]
	}

	return value.NewValue(ra), nil
}

/*
The constructor returns a NewObjectValues with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *ObjectValues) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectValues(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectAdd
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_ADD(expr, expr, expr).
It returns an object containing the source object augmented
with the new name, attribute pair.
It does not do key substitution.
Type ObjectAdd is a struct that implements TernaryFunctionBase.
*/
type ObjectAdd struct {
	TernaryFunctionBase
}

/*
The function NewObjectAdd calls NewTernaryFunctionBase to
create a function named OBJECT_PUT with three expression as
input.
*/
func NewObjectAdd(first, second, third Expression) Function {
	rv := &ObjectAdd{
		*NewTernaryFunctionBase("object_add", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ObjectAdd) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type OBJECT.
*/
func (this *ObjectAdd) Type() value.Type { return value.OBJECT }

/*
Calls the Eval method for ternary functions and passes in the
receiver, current item and current context.
*/
func (this *ObjectAdd) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

/*
This method takes in an object, a name and a value
and returns a new object that contains the name /
attribute pair. If the type of input is missing
then return a missing value, and if not an object
return a null value.
If the key is found, an error is thrown
*/
func (this *ObjectAdd) Apply(context Context, first, second, third value.Value) (value.Value, error) {

	// First must be an object, or we're out
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	// second must be a non empty string
	if second.Type() != value.STRING || second.Actual().(string) == "" {
		return first, nil
	}

	field := second.Actual().(string)

	// we don't overwrite
	_, exists := first.Field(field)
	if exists {
		return value.NULL_VALUE, nil
	}

	// SetField will remove if the attribute is missing, but we don't
	// overwrite anyway, so we might just skip now
	if third.Type() != value.MISSING {
		rv := first.CopyForUpdate()
		rv.SetField(field, third)
		return rv, nil
	}
	return first, nil
}

/*
The constructor returns a NewObjectAdd with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *ObjectAdd) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectAdd(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectPut
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_PUT(expr, expr, expr).
It returns an object containing the source object augmented
with the new name, attribute pair.
If the key is found in the object, the corresponding attribute
is replaced by the third argument.
If the third argument is MISSING, the existing key is deleted.
Type ObjectPut is a struct that implements UnaryFunctionBase.
*/
type ObjectPut struct {
	TernaryFunctionBase
}

/*
The function NewObjectPut calls NewTernaryFunctionBase to
create a function named OBJECT_PUT with three expression as
input.
*/
func NewObjectPut(first, second, third Expression) Function {
	rv := &ObjectPut{
		*NewTernaryFunctionBase("object_put", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ObjectPut) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type OBJECT.
*/
func (this *ObjectPut) Type() value.Type { return value.OBJECT }

/*
Calls the Eval method for ternary functions and passes in the
receiver, current item and current context.
*/
func (this *ObjectPut) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

/*
This method takes in an object, a name and a value
and returns a new object that contains the name /
attribute pair. If the type of input is missing
then return a missing value, and if not an object
return a null value.
If the key passed already exists, then the attribute
replaces the old attribute. If the attribute is missing
this function behaves like OBJECT_REMOVE
*/
func (this *ObjectPut) Apply(context Context, first, second, third value.Value) (value.Value, error) {

	// First must be an object, or we're out
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	// second must be a non empty string
	if second.Type() != value.STRING || second.Actual().(string) == "" {
		return first, nil
	}

	field := second.Actual().(string)

	rv := first.CopyForUpdate()
	rv.SetField(field, third)
	return rv, nil
}

/*
The constructor returns a NewObjectPut with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *ObjectPut) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectPut(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectRemove
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_REMOVE(expr, expr).
It returns an object with the name / attribute pair for
the name passed as second parameter removed.
Type ObjectRemove is a struct that implements BinaryFunctionBase.
*/
type ObjectRemove struct {
	BinaryFunctionBase
}

/*
The function NewObjectRemove calls NewBinaryFunctionBase to
create a function named OBJECT_REMOVE with two expressions as
input.
*/
func NewObjectRemove(first, second Expression) Function {
	rv := &ObjectRemove{
		*NewBinaryFunctionBase("object_remove", first, second),
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *ObjectRemove) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value type OBJECT.
*/
func (this *ObjectRemove) Type() value.Type { return value.OBJECT }

/*
Calls the Eval method for binary functions and passes in the
receiver, current item and current context.
*/
func (this *ObjectRemove) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
}

/*
This method takes in an object and a name and returns
an object with the name / attribute pair removed.
If the type of input is missing then return a missing value, and
if not an object return a null value.
*/
func (this *ObjectRemove) Apply(context Context, first, second value.Value) (value.Value, error) {
	// First must be an object, or we're out
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	// second must be a non empty string
	if second.Type() != value.STRING || second.Actual().(string) == "" {
		return first, nil
	}

	field := second.Actual().(string)

	rv := first.CopyForUpdate()
	rv.UnsetField(field)
	return rv, nil
}

/*
The constructor returns a NewObjectRemove with the an operand
cast to a Function as the FunctionConstructor.
*/
func (this *ObjectRemove) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectRemove(operands[0], operands[1])
	}
}
