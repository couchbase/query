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

	"github.com/couchbaselabs/query/value"
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
