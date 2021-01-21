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
	"math"
	"sort"

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

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

func NewObjectAdd(first, second, third Expression) Function {
	rv := &ObjectAdd{
		*NewTernaryFunctionBase("object_add", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectAdd) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectAdd) Type() value.Type { return value.OBJECT }

func (this *ObjectAdd) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *ObjectAdd) PropagatesMissing() bool {
	return false
}

func (this *ObjectAdd) PropagatesNull() bool {
	return false
}

/*
This method takes in an object, a name and a value and returns a new
object that contains the name / attribute pair. If the first input is
missing then return a missing value, and if not an object return a
null value.
*/
func (this *ObjectAdd) Apply(context Context, first, second, third value.Value) (value.Value, error) {

	// Check for type mismatches
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.OBJECT || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	field := second.Actual().(string)

	// we don't overwrite
	_, exists := first.Field(field)
	if exists {
		return first, nil
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
Factory method pattern.
*/
func (this *ObjectAdd) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectAdd(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectConcat
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_CONCAT(expr1, expr2 ...).
It returns a new object with the concatenation of the input
objects.
*/
type ObjectConcat struct {
	FunctionBase
}

func NewObjectConcat(operands ...Expression) Function {
	rv := &ObjectConcat{
		*NewFunctionBase("object_concat", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectConcat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectConcat) Type() value.Type { return value.OBJECT }

func (this *ObjectConcat) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() != value.OBJECT {
			null = true
		} else if !null && !missing {
			if i == 0 {
				rv = arg.CopyForUpdate()
			} else {
				fields := arg.Fields()
				for n, v := range fields {
					rv.SetField(n, v)
				}
			}
		}

	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	return rv, nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ObjectConcat) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ObjectConcat) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ObjectConcat) Constructor() FunctionConstructor {
	return NewObjectConcat
}

///////////////////////////////////////////////////
//
// ObjectInnerPairs
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_INNERPAIRS(expr).
It returns an array containing the attribute name and
value pairs of the object, in N1QL collation order of
the names.
*/
type ObjectInnerPairs struct {
	UnaryFunctionBase
}

func NewObjectInnerPairs(operand Expression) Function {
	rv := &ObjectInnerPairs{
		*NewUnaryFunctionBase("object_innerpairs", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectInnerPairs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectInnerPairs) Type() value.Type { return value.ARRAY }

/*
This method takes in an object and returns a map of name
value pairs. If the type of input is missing then return
a missing value, and if not an object return a null value.
Convert it to a valid Go type. Cast it to a map from
string to interface. Range over this map and save the keys.
Sort the keys and range over the keys to create name and value
pairs. Return this object.
*/
func (this *ObjectInnerPairs) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := removeMissing(arg)

	var nameBuf [_NAME_CAP]string
	var names []string
	if len(oa) <= len(nameBuf) {
		names = nameBuf[0:0]
	} else {
		names = _NAME_POOL.GetCapped(len(oa))
		defer _NAME_POOL.Put(names)
	}

	for name, _ := range oa {
		names = append(names, name)
	}

	sort.Strings(names)
	ra := make([]interface{}, len(names))
	for i, n := range names {
		ra[i] = map[string]interface{}{"name": n, "val": oa[n]}
	}

	return value.NewValue(ra), nil
}

/*
Factory method pattern.
*/
func (this *ObjectInnerPairs) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectInnerPairs(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectInnerValues
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_INNERVALUES(expr).
It returns an array containing the attribute values of
the object, in N1QL collation order.
*/
type ObjectInnerValues struct {
	UnaryFunctionBase
}

func NewObjectInnerValues(operand Expression) Function {
	rv := &ObjectInnerValues{
		*NewUnaryFunctionBase("object_innervalues", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectInnerValues) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectInnerValues) Type() value.Type { return value.ARRAY }

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
func (this *ObjectInnerValues) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := removeMissing(arg)

	var nameBuf [_NAME_CAP]string
	var names []string
	if len(oa) <= len(nameBuf) {
		names = nameBuf[0:0]
	} else {
		names = _NAME_POOL.GetCapped(len(oa))
		defer _NAME_POOL.Put(names)
	}

	for name, _ := range oa {
		names = append(names, name)
	}

	sort.Strings(names)
	ra := make([]interface{}, len(names))
	for i, n := range names {
		ra[i] = oa[n]
	}

	return value.NewValue(ra), nil
}

/*
Factory method pattern.
*/
func (this *ObjectInnerValues) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectInnerValues(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectLength
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_LENGTH(expr).
It returns the number of name-value pairs in the object.
*/
type ObjectLength struct {
	UnaryFunctionBase
}

func NewObjectLength(operand Expression) Function {
	rv := &ObjectLength{
		*NewUnaryFunctionBase("object_length", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectLength) Type() value.Type { return value.NUMBER }

func (this *ObjectLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	return value.NewValue(len(oa)), nil
}

/*
Factory method pattern.
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
the object, in N1QL collation order.
*/
type ObjectNames struct {
	UnaryFunctionBase
}

func NewObjectNames(operand Expression) Function {
	rv := &ObjectNames{
		*NewUnaryFunctionBase("object_names", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectNames) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectNames) Type() value.Type { return value.ARRAY }

func (this *ObjectNames) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})

	var nameBuf [_NAME_CAP]string
	var names []string
	if len(oa) <= len(nameBuf) {
		names = nameBuf[0:0]
	} else {
		names = _NAME_POOL.GetCapped(len(oa))
		defer _NAME_POOL.Put(names)
	}

	for name, _ := range oa {
		names = append(names, name)
	}

	sort.Strings(names)
	ra := make([]interface{}, len(names))
	for i, n := range names {
		ra[i] = n
	}

	return value.NewValue(ra), nil
}

/*
Factory method pattern.
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
the names.
*/
type ObjectPairs struct {
	UnaryFunctionBase
}

func NewObjectPairs(operand Expression) Function {
	rv := &ObjectPairs{
		*NewUnaryFunctionBase("object_pairs", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectPairs) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectPairs) Type() value.Type { return value.ARRAY }

/*
This method takes in an object and returns a map of name
value pairs. If the type of input is missing then return
a missing value, and if not an object return a null value.
Convert it to a valid Go type. Cast it to a map from
string to interface. Range over this map and save the keys.
Sort the keys and range over the keys to create name and value
pairs. Return this object.
*/
func (this *ObjectPairs) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})

	var nameBuf [_NAME_CAP]string
	var names []string
	if len(oa) <= len(nameBuf) {
		names = nameBuf[0:0]
	} else {
		names = _NAME_POOL.GetCapped(len(oa))
		defer _NAME_POOL.Put(names)
	}

	for name, _ := range oa {
		names = append(names, name)
	}

	sort.Strings(names)
	ra := make([]interface{}, len(names))
	for i, n := range names {
		ra[i] = map[string]interface{}{"name": n, "val": oa[n]}
	}

	return value.NewValue(ra), nil
}

/*
Factory method pattern.
*/
func (this *ObjectPairs) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectPairs(operands[0])
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
*/
type ObjectPut struct {
	TernaryFunctionBase
}

func NewObjectPut(first, second, third Expression) Function {
	rv := &ObjectPut{
		*NewTernaryFunctionBase("object_put", first, second, third),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectPut) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectPut) Type() value.Type { return value.OBJECT }

func (this *ObjectPut) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *ObjectPut) PropagatesMissing() bool {
	return false
}

func (this *ObjectPut) PropagatesNull() bool {
	return false
}

/*
This method takes in an object, a name and a value
and returns a new object that contains the name /
attribute pair. If the type of input is missing
then return a missing value, and if not an object
return a null value.
If the key passed already exists, then the attribute
replaces the old attribute. If the attribute is missing
this function behaves like OBJECT_REMOVE.
*/
func (this *ObjectPut) Apply(context Context, first, second, third value.Value) (value.Value, error) {

	// Check for type mismatches
	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.OBJECT || second.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	field := second.Actual().(string)

	rv := first.CopyForUpdate()
	rv.SetField(field, third)
	return rv, nil
}

/*
Factory method pattern.
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
This represents the object function OBJECT_REMOVE(expr, name ...).  It
returns an object with the name / attribute pair for the name passed
as second parameter removed.
*/
type ObjectRemove struct {
	FunctionBase
}

func NewObjectRemove(operands ...Expression) Function {
	rv := &ObjectRemove{
		*NewFunctionBase("object_remove", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectRemove) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectRemove) Type() value.Type { return value.OBJECT }

/*
This method takes in an object and names and returns
an object with the name / attribute pairs removed.
If the type of input is missing then return a missing value, and
if not an object return a null value.
*/
func (this *ObjectRemove) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	null := false
	missing := false
	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() == value.NULL {
			null = true
		} else if !null && !missing {
			if i == 0 {
				if arg.Type() != value.OBJECT {
					null = true
				} else {
					rv = arg.CopyForUpdate()
				}
			} else {
				if arg.Type() != value.STRING {
					null = true
				} else {
					n := arg.Actual().(string)
					rv.UnsetField(n)
				}
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	return rv, nil
}

/*
Minimum input arguments required is 2.
*/
func (this *ObjectRemove) MinArgs() int { return 2 }

/*
Maximum input arguments allowed.
*/
func (this *ObjectRemove) MaxArgs() int { return math.MaxInt16 }

/*
Factory method pattern.
*/
func (this *ObjectRemove) Constructor() FunctionConstructor {
	return NewObjectRemove
}

///////////////////////////////////////////////////
//
// ObjectRename
//
///////////////////////////////////////////////////

/*
This represents the function OBJECT_RENAME(obj, old_name, new_name).
Returns a new object with the name old_name replaced by new_name.
*/
type ObjectRename struct {
	TernaryFunctionBase
}

func NewObjectRename(obj, old_name, new_name Expression) Function {
	rv := &ObjectRename{
		*NewTernaryFunctionBase("object_rename", obj, old_name, new_name),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectRename) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectRename) Type() value.Type { return value.OBJECT }

func (this *ObjectRename) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *ObjectRename) Apply(context Context, obj, old_name, new_name value.Value) (
	value.Value, error) {

	// Check for type mismatches
	if obj.Type() == value.MISSING || old_name.Type() == value.MISSING || new_name.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if obj.Type() != value.OBJECT || old_name.Type() != value.STRING || new_name.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	old := old_name.Actual().(string)
	val, ok := obj.Field(old)
	if !ok {
		return obj, nil
	}

	rv := obj.CopyForUpdate()
	rv.UnsetField(old)
	rv.SetField(new_name.Actual().(string), val)
	return rv, nil
}

/*
Factory method pattern.
*/
func (this *ObjectRename) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectRename(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectReplace
//
///////////////////////////////////////////////////

/*
This represents the function OBJECT_REPLACE(obj, old_val,
new_val).  Returns a new object with all occurrences of old_val
replaced by new_val.
*/
type ObjectReplace struct {
	TernaryFunctionBase
}

func NewObjectReplace(obj, old_val, new_val Expression) Function {
	rv := &ObjectReplace{
		*NewTernaryFunctionBase("object_replace", obj, old_val, new_val),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectReplace) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectReplace) Type() value.Type { return value.OBJECT }

func (this *ObjectReplace) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.TernaryEval(this, item, context)
}

func (this *ObjectReplace) PropagatesMissing() bool {
	return false
}

func (this *ObjectReplace) PropagatesNull() bool {
	return false
}

func (this *ObjectReplace) Apply(context Context, obj, old_val, new_val value.Value) (
	value.Value, error) {

	// Check for type mismatches
	if obj.Type() == value.MISSING || old_val.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if obj.Type() != value.OBJECT || old_val.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	dup := obj.CopyForUpdate()
	fields := dup.Fields()
	for name, val := range fields {
		if old_val.Equals(value.NewValue(val)).Truth() {
			dup.SetField(name, new_val)
		}
	}

	return dup, nil
}

/*
Factory method pattern.
*/
func (this *ObjectReplace) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectReplace(operands[0], operands[1], operands[2])
	}
}

///////////////////////////////////////////////////
//
// ObjectUnwrap
//
///////////////////////////////////////////////////

/*
This represents the object function OBJECT_UNWRAP(expr).
Given an object with precisely one name / value pair, it
returns the value.
*/
type ObjectUnwrap struct {
	UnaryFunctionBase
}

func NewObjectUnwrap(operand Expression) Function {
	rv := &ObjectUnwrap{
		*NewUnaryFunctionBase("object_unwrap", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectUnwrap) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectUnwrap) Type() value.Type { return value.JSON }

/*
This method takes in an object and returns the
attribute value. If the type of input is missing
then return a missing value, and if not an object
return a null value.
*/
func (this *ObjectUnwrap) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	if len(oa) == 1 {
		for _, v := range oa {
			return value.NewValue(v), nil
		}
	}

	return value.NULL_VALUE, nil
}

/*
Factory method pattern.
*/
func (this *ObjectUnwrap) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectUnwrap(operands[0])
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
names.
*/
type ObjectValues struct {
	UnaryFunctionBase
}

func NewObjectValues(operand Expression) Function {
	rv := &ObjectValues{
		*NewUnaryFunctionBase("object_values", operand),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectValues) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectValues) Type() value.Type { return value.ARRAY }

/*
This method takes in an object and returns a slice
that contains the attribute values. If the type of
input is missing then return a missing value, and
if not an object return a null value.
*/
func (this *ObjectValues) Evaluate(item value.Value, context Context) (value.Value, error) {
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})

	var nameBuf [_NAME_CAP]string
	var names []string
	if len(oa) <= len(nameBuf) {
		names = nameBuf[0:0]
	} else {
		names = _NAME_POOL.GetCapped(len(oa))
		defer _NAME_POOL.Put(names)
	}

	for name, _ := range oa {
		names = append(names, name)
	}

	sort.Strings(names)
	ra := make([]interface{}, len(names))
	for i, n := range names {
		ra[i] = oa[n]
	}

	return value.NewValue(ra), nil
}

/*
Factory method pattern.
*/
func (this *ObjectValues) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectValues(operands[0])
	}
}

/*
Utility function to remove missing array elements for OBJECT_INNERVALUES
and OBJECT_INNERPAIRS
*/
func removeMissing(arg value.Value) map[string]interface{} {
	if len(arg.Actual().(map[string]interface{})) == 1 {
		return arg.Actual().(map[string]interface{})
	}

	oa := arg.Copy().Actual().(map[string]interface{})
	for name, val := range oa {
		valSlice, ok := val.([]interface{})
		if !ok {
			continue
		}
		newSlice := make([]interface{}, 0, len(valSlice))
		for _, subVal := range valSlice {
			if value.NewValue(subVal).Type() != value.MISSING {
				newSlice = append(newSlice, subVal)
			}
		}
		if len(newSlice) == 1 {
			oa[name] = newSlice[0]
		} else {
			oa[name] = newSlice
		}
	}
	return oa
}

const _NAME_CAP = 16

var _NAME_POOL = util.NewStringPool(256)
