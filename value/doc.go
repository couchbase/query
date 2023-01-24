/*
Copyright 2014-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

/*
	## Source code distribution :

###Note:
- Any definition starting with a capital letter is public; the small case definitions can be seen only by that particular package.
- Both shallow and deep copy of a non-array/object value is that value itself.
-A copy should give you a copy that you can use in another thread.

value.go: The first constant declaration is _Tristate_.  It has three states, _TRUE_, _FALSE_ and _NONE_ (initialized to _iota_, which is used in constant declarations in Go. Its value starts at 0 in the const block and increments each time it is seen.  Refer to the Golang docs). Value supports different data types that are supported by N1ql. They are Missing, Null, Binary, Boolean, Number, String, Array, Object and JSON. The _MARSHAL_ERROR constant represents an error string that is output when there is an unexpected marshal error on valid data. Marshal returns the JSON encoding of any input interface. It is used while implementing the method MarshalJSON. The type ValueChannel is a channel of Value, where Value is an interface. (Channels in Go are pipes that connect concurrent goroutines) The results in a request for the server are of type ValueChannel. Values is a collection of Value. The type is defined as a slice of Value. Similarly CompositeValues is a slice of Values.

Before we discuss the Value interface it is important to note the definition of the type Unsettable as a string.  This is used to report an error when an input type is incorrectly trying to set a nested property (valid in the case of an object) or index (valid in the case of an array) that does not exist for it, by invoking one of these methods defined in the Value interface.  The method receiver (a specially defined first argument for a function that is an instance of the type that it refers to, for eg. this Unsettable) is of type Unsettable and the method returns a string. If it is an empty string then it returns “Not Settable”, and if not then it specifies the field or index that is not settable which was an argument to the SetField or SetIndex method. Type Value is an interface that is used for storing and manipulating a JSON value. It has a list of methods that are discussed below.  Each value will implement the methods that correspond to it, as discussed in the following sections.

Before we discuss the methods Slice and SliceTail it is important to be familiar with what a Slice is in Go.  Slices are similar to arrays in their underlying representation and have no specified length. It is a descriptor of an array segment and consists of a pointer to the array, the length of the slice, and the capacity of the slice (For e.g. we could allocate an array of 5 elements and create a slice from 2:4, the length of the slice in this case is 2 and the capacity is 3). It can be created with the built in make function where we can specify the length and capacity.  The syntax for defining a slice allows you to leave the start and/or end index blank. (For e.g. a[:], a[2:], a[:5]). It is important to note that in N1QL, the start of a slice has to be specified.For more detailed information please refer to the Golang documents chapter on arrays and slices.
In order to understand the default behaviour of the NewValue function better it is important to be familiar with the concept of reflection. It is a languages ability to inspect and dynamically call methods defined for that type at runtime and is primarily useful by statically typed languages (since if it was dynamically typed the compiler will allow the method to be called by any object failing at runtime if it doesn’t exist).  To read more implementation specific details about reflection in Go please refer to the golang laws of reflection.

missing.go: Type missingValue is a type string. Variable MISSING_VALUE is defined and initialized to an empty string cast to missingValue.  The function NewMissingValue() returns a value, by returning MISSING_VALUE. The methods implemented for this value have a method receiver of type missingValue. The MarshalJSON implementation for this value, returns ‘_NULL_BYTES’, defined in value/null.go and nil since we should never marshal a missing value (if is not a valid json type). Type missingValue implements all the methods defined by the Value interface. Please refer to the table that compares Null and Missing values to better understand Equals and Collate.

null.go: nullValue type is an empty struct. The variable NULL_VALUE is initialized as a pointer to an empty nullValue.  The variable _NULL_BYTES is a slice of byte representing null string. The NewNullValue function returns a value NULL_VALUE. The implemented methods for null are similar to those for missing.

boolean.go: Represented by boolValue.

number.go: floatValue is defined as type float64. (There are other types such as int, float32 etc, but they are not used at this moment.)

string.go: stringValue is defined as type string. The major difference is for the method Collate, when the type of input argument is stringValue. Here we compare the 2 strings and return -1 if the receiver is less than the input.

array.go: sliceValue is defined as a slice of interfaces. Methods that deal with the Field are not valid for arrays and hence return Unsettable. Since sliceValue defined slices do not extend beyond the set length, we create a new type listValue that is a struct containing slice values. This enables us to call all the implemented methods for slicevalue without having to redefine them.

object.go :vobjectValue is a type of map from string to interface. For the SetField method, the reason that delete is called as opposed to calling the function UnsetField is in the future, for this function, we might decide to throw an error in the event that the field was missing.

annotated.go: The annotatedValue is used to handle any extra information about the value, in the form of metadata.

scope.go: ScopeValue provides alias scoping for subqueries, for’s, lets etc.  ScopeValue is a type struct that inherits Value and has a parent Value.

sort.go: The type Sorter is a structure containing one element of type Value. It sorts an array value in place.

parsed.go: It is a structure for storing and manipulating values. Use the the fast key and index methods in go_json.

set.go: Set implements a hash set of values.
*/
package value
