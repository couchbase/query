//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"bytes"
	"encoding/json"
	"sort"
)

/*
objectValue is a type of map from string to interface.
*/
type objectValue map[string]interface{}

/*
The method MarshalJSON checks to see if the receiver of type
objectValue is nil and if so returns _NULL _BYTES. If not it
creates a new buffer and writes a ’{‘ to it. We call function
sortedNames on the receiver to sort the fields. It uses the
Sort package to sort the keys in the object.  We range over
all the keys, for each value associated with the keys, if its
type is missing do not populate the field. If not and the
iterator is greater than 0, add a ‘,’ to the buffer (since
that means 1 field has been populated in the buffer). Write
out a ‘ ” ’, then the key, a ‘ : ‘, and then the value.
Before the value is written out to the buffer, Marshal it
and check for errors. Finally once all the fields of the
object have been marshaled  ‘}’  is written to the buffer and
the bytes are returned by calling Bytes(). This is in keeping
with the JSON format to define objects.
*/
func (this objectValue) MarshalJSON() ([]byte, error) {
	if this == nil {
		return _NULL_BYTES, nil
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1<<8))
	buf.WriteString("{")

	names := sortedNames(this)
	for i, n := range names {
		v := NewValue(this[n])
		if v.Type() == MISSING {
			continue
		}

		if i > 0 {
			buf.WriteString(",")
		}

		buf.WriteString("\"")
		buf.WriteString(n)
		buf.WriteString("\":")

		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		buf.Write(b)
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

/*
Type OBJECT.
*/
func (this objectValue) Type() Type { return OBJECT }

/*
Return receiver by casting it to a map of string to interfaces.
*/
func (this objectValue) Actual() interface{} {
	return (map[string]interface{})(this)
}

/*
Return true if objects are equal, else false (refer N1QL specs).
For internal types *scopevalue, *annotatedvalue and *parsedvalue
call Equals again on the value of other. (For parsed value parse
the input before calling equals). For type objectvalue call
objectEquals to do an element by element comparison.
*/
func (this objectValue) Equals(other Value) bool {
	switch other := other.(type) {
	case objectValue:
		return objectEquals(this, other)
	case *ScopeValue:
		return this.Equals(other.Value)
	case *annotatedValue:
		return this.Equals(other.Value)
	case *parsedValue:
		return this.Equals(other.parse())
	default:
		return false
	}
}

/*
Return int representing position of object with other values.
For *scopeValue, *annotatedValue and *parsedValue call Collate
on the value of other. (For parsed value parse the input
before calling Collate). For type objectvalue call
objectCollate to determine object ordering.
*/
func (this objectValue) Collate(other Value) int {
	switch other := other.(type) {
	case objectValue:
		return objectCollate(this, other)
	case *ScopeValue:
		return this.Collate(other.Value)
	case *annotatedValue:
		return this.Collate(other.Value)
	case *parsedValue:
		return this.Collate(other.parse())
	default:
		return 1
	}
}

/*
If length of the object is greater than 0 return true.
It needs to have a minimum of 1 element.
*/
func (this objectValue) Truth() bool {
	return len(this) > 0
}

/*
It calls copyMap with inputs this (the receiver) and self.
*/
func (this objectValue) Copy() Value {
	return objectValue(copyMap(this, self))
}

/*
It calls copyMap with inputs this (the receiver) and
copyForUpdate.
*/
func (this objectValue) CopyForUpdate() Value {
	return objectValue(copyMap(this, copyForUpdate))
}

/*
It returns a field in an object. It initializes result
to the field in this (it is a map and hence accesses
that value). If ok then return the result by converting
it into the Value type system and return true for the bool.
If the field does not exist in the object then it returns
a missing and a false to indicate that the field was not
found. It does this by calling missingField on the field.
*/
func (this objectValue) Field(field string) (Value, bool) {
	result, ok := this[field]
	if ok {
		return NewValue(result), true
	}

	return missingField(field), false
}

/*
The SetField method returns an error that depicts if the
field was successfully set. The method receiver is of
type objectValue and the function returns an error stating
if the field of type string was successfully set and mapped
to the val of type interface. The code checks to see the
type of value, if it is a missingValue it deletes the field,
but the default behavior is to set the value for the field
in the map that defines objectValue.
*/
func (this objectValue) SetField(field string, val interface{}) error {
	switch val := val.(type) {
	case missingValue:
		delete(this, field)
	default:
		this[field] = val
	}

	return nil
}

/*
The UnsetField method takes the field string as an input
and gives you an error. It takes as input the field to
delete, and deletes it from the object. It returns nil to
indicate that the specified field has been deleted successfully.
*/
func (this objectValue) UnsetField(field string) error {
	delete(this, field)
	return nil
}

/*
Calls missingIndex.
*/
func (this objectValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Not valid for objects.
*/
func (this objectValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

/*
Returns NULL_VALUE.
*/
func (this objectValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns NULL_VALUE.
*/
func (this objectValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

/*
It flattens out the elements of the object and appends it into
the buffer. This is done in child first (depth first) order.
In the event the buffer is full (capacity < length of the
buffer + the current element), then grow the buffer by
twice of length of the buffer + this element + 1.  Once the
buffer has space,range over the objects, sort over all the
fields and then append the children values to the buffer,
and call Descendants recursively until there are no elements
left. Finally return the buffer.
*/
func (this objectValue) Descendants(buffer []interface{}) []interface{} {
	names := sortedNames(this)

	if cap(buffer) < len(buffer)+len(this) {
		buf2 := make([]interface{}, len(buffer), (len(buffer)+len(this)+1)<<1)
		copy(buf2, buffer)
		buffer = buf2
	}

	for _, name := range names {
		val := this[name]
		buffer = append(buffer, val)
		buffer = NewValue(val).Descendants(buffer)
	}

	return buffer
}

/*
Return the receiver this.
*/
func (this objectValue) Fields() map[string]interface{} {
	return this
}

/*
Do an element by element comparison to return true if all
fields are the same and false if not. The first comparison
made by the function is the length of the two objects, if
not the same it returns false. Range over the first object.
If the value of the second is not equal to the value of the
first (note that they are already in sorted order)or if that
field is missing then return false. If not return true which
means that all the fields match and the two objects are equal.
*/
func objectEquals(obj1, obj2 map[string]interface{}) bool {
	if len(obj1) != len(obj2) {
		return false
	}

	for name1, val1 := range obj1 {
		val2, ok := obj2[name1]
		if !ok || !NewValue(val1).Equals(NewValue(val2)) {
			return false
		}
	}

	return true
}

/*
This code originally taken from https://github.com/couchbaselabs/walrus.
Used to determine object ordering.  The function takes two
objects (both maps) and returns an int. The first step is
to see if one object is larger than the other and directly
return that difference. If lengths are equal, do a
name-by-name comparison.  The first step is to collect all
the keys (field names) in the object and initialize their
values to false. Range over all the fields and compare the
values associated with them by calling collate (if there
was no corresponding value in the objects map under a field
then return 1 if the field was missing in object 1 and
-1 if it was missing in field2. This is as per the N1QL specs).
If Collate returns a non-zero value that is returned. If it
is zero, continue to compare the rest of the fields.
Finally since all the names and values are equal, return 0.
*/
func objectCollate(obj1, obj2 map[string]interface{}) int {
	// first see if one object is larger than the other
	delta := len(obj1) - len(obj2)
	if delta != 0 {
		return delta
	}

	// if not, proceed to do name by name comparision

	// collect all the names
	allmap := make(map[string]bool, len(obj1)<<1)
	for n, _ := range obj1 {
		allmap[n] = false
	}
	for n, _ := range obj2 {
		allmap[n] = false
	}

	allnames := make(sort.StringSlice, len(allmap))
	i := 0
	for n, _ := range allmap {
		allnames[i] = n
		i++
	}

	// sort the names
	allnames.Sort()

	// now compare the values associated with each name
	for _, name := range allnames {
		val1, ok := obj1[name]
		if !ok {
			// obj1 did not have this name, so it is larger
			return 1
		}

		val2, ok := obj2[name]
		if !ok {
			// ojb2 did not have this name, so it is larger
			return -1
		}

		// name was in both objects, so compare the corresponding values
		cmp := NewValue(val1).Collate(NewValue(val2))
		if cmp != 0 {
			return cmp
		}
	}

	// all names and values are equal
	return 0
}

/*
It allows for a copy of every field in the object by using a copyFunc.
If the source is nil then return nil. If not create a result
map, range over the source and add it into the result by casting
it to the copier. Once this is done return the result.
*/
func copyMap(source map[string]interface{}, copier copyFunc) map[string]interface{} {
	if source == nil {
		return nil
	}

	result := make(map[string]interface{}, len(source))
	for n, v := range source {
		result[n] = copier(v)
	}

	return result
}

/*
Takes an input a map and returns a string that represents a sorted
set of keys. Range over the object and append all keys to a variable
that defined as a slice of strings. The type StringSlice is defined
by the sort package and its sort method sorts in increasing order
of input. Sort this slice of string and return it back to the caller
(namely MarshalJSON).
*/
func sortedNames(obj map[string]interface{}) []string {
	names := make(sort.StringSlice, 0, len(obj))
	for name, _ := range obj {
		names = append(names, name)
	}

	names.Sort()
	return names
}
