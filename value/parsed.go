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
	"fmt"
	"strconv"

	jsonpointer "github.com/dustin/go-jsonpointer"
)

/*
A structure for storing and manipulating a value. It contains
three elements. The first is raw which is a slice of bytes,
the second is parsedType which is of type Type, and finally
parsed which is a Value.
*/
type parsedValue struct {
	raw        []byte
	parsedType Type
	parsed     Value
}

/*
Check for the receivers parsedType. If it is an object or
array, call its method MarshalJSON. If it is binary then
return the raw bytes and an error saying that Marshaling
a binary value returns raw bytes. The default is to return
the raw bytes and nil as the error.
*/
func (this *parsedValue) MarshalJSON() ([]byte, error) {
	switch this.parsedType {
	case OBJECT, ARRAY:
		return this.parse().MarshalJSON()
	case BINARY:
		return this.raw, fmt.Errorf("Marshaling binary value returns raw bytes: %v", string(this.raw))
	default:
		return this.raw, nil
	}
}

/*
Return the parsedType of the receiver.
*/
func (this *parsedValue) Type() Type { return this.parsedType }

/*
Check if parsedType is binary. If it is then return the raw
bytes. Otherwise call the Actual method for the values in
the receiver.
*/
func (this *parsedValue) Actual() interface{} {
	if this.parsedType == BINARY {
		return this.raw
	}

	return this.parse().Actual()
}

/*
Checks if the raw bytes in *parsedValue are equal to the input
Value. Check to see of the parsedType is binary. If it is
marshal the second value and call bytes.Equal to check if the
bytes are equal. If not binary, parse it first and then call
Equals again. The parsedValue has raw bytes and it will
eventually be parsed. It implements delayed parsing.
*/
func (this *parsedValue) Equals(other Value) bool {
	if this.parsedType == BINARY {
		b, _ := other.MarshalJSON()
		return bytes.Equal(this.raw, b)
	}

	return this.parse().Equals(other)
}

/*
If the parsedType for the receiver is binary, and the
other value type is also binary then call MarshalJSON
on the other value and do a bytes compare. If the other
is not of type binary, return the relative position of
that type with respect to binary. Finally if the receiver
type was not binary parse it, and then call collate again.
*/
func (this *parsedValue) Collate(other Value) int {
	if this.parsedType == BINARY {
		if other.Type() == BINARY {
			b, _ := other.MarshalJSON()
			return bytes.Compare(this.raw, b)
		} else {
			return int(BINARY - other.Type())
		}
	}

	return this.parse().Collate(other)
}

/*
Return true if the recievers parsedType is Binary. If not
,parse the input bytes and then call Truth on it.
*/
func (this *parsedValue) Truth() bool {
	if this.parsedType == BINARY {
		return true
	}

	return this.parse().Truth()
}

/*
Check if the parsedtype is not binary and it isnt an array
or object then parse the bytes and call that values
respective Copy method. If it is binary then set the raw
variable for a struct to the receivers raw value and
the parsedType to the receivers parsedType and return a
pointer to this struct.
*/
func (this *parsedValue) Copy() Value {
	if this.parsedType != BINARY && this.parsedType < ARRAY {
		return this.parse().Copy()
	}

	return &parsedValue{
		raw:        this.raw,
		parsedType: this.parsedType,
	}
}

/*
If the receivers parsedType is Binary, call the Copy function
and return. If not, parse the bytes and then call CopyForUpdate
over that.
*/
func (this *parsedValue) CopyForUpdate() Value {
	if this.parsedType == BINARY {
		return this.Copy()
	}

	return this.parse().CopyForUpdate()
}

/*
Use "github.com/dustin/go-jsonpointer".
First check if the parsed Value is nil. If not then call the
Field method for that value and return it. In the event that
it is nil, check to see if the parsedType is an object. If not
return missingField and false, since only objects have fields.
Use the Find method in the jsonpointer package to find a section
of raw JSON, with input arguments the slice of bytes and a path
string. The package defines a string syntax for indentifying a
specific value in a JSON document. It returns a slice of bytes.
If the error it returns is not nil or if the result of the Find
is nil and the error is nil, then return a missingField.
If the result is not nil then call NewValue on the result to
get a valid value and return true.
*/
func (this *parsedValue) Field(field string) (Value, bool) {
	if this.parsed != nil {
		return this.parsed.Field(field)
	}

	if this.parsedType != OBJECT {
		return missingField(field), false
	}

	res, err := jsonpointer.Find(this.raw, "/"+field)
	if err != nil {
		return missingField(field), false
	}
	if res != nil {
		return NewValue(res), true
	}

	return missingField(field), false
}

/*
Return Unsettable if parsedType is not OBJECT. If it is then parse
the receiver and call the values corresponding SetField.
*/
func (this *parsedValue) SetField(field string, val interface{}) error {
	if this.parsedType != OBJECT {
		return Unsettable(field)
	}

	return this.parse().SetField(field, val)
}

/*
Return Unsettable if parsedType is not OBJECT. If it is then parse
the receiver and call the values corresponding UnsetField.
*/
func (this *parsedValue) UnsetField(field string) error {
	if this.parsedType != OBJECT {
		return Unsettable(field)
	}

	return this.parse().UnsetField(field)
}

/*
Call the index method for the type of value parsed, if it is
not nil. If it isnt of type array then return missingIndex.
Go through the raw bytes and find this index. If there is an
error or the result is nil, return missingIndex. Otherwise
call NewValue to get a value to return.
*/
func (this *parsedValue) Index(index int) (Value, bool) {
	if this.parsed != nil {
		return this.parsed.Index(index)
	}

	if this.parsedType != ARRAY {
		return missingIndex(index), false
	}

	if this.raw != nil {
		res, err := jsonpointer.Find(this.raw, "/"+strconv.Itoa(index))
		if err != nil {
			return missingIndex(index), false
		}
		if res != nil {
			return NewValue(res), true
		}
	}

	return missingIndex(index), false
}

/*
Return Unsettable if parsedType is not ARRAY. If it is then parse
the receiver and call the values corresponding SetIndex with the
index and value as input arguments.
*/
func (this *parsedValue) SetIndex(index int, val interface{}) error {
	if this.parsedType != ARRAY {
		return Unsettable(index)
	}

	return this.parse().SetIndex(index, val)
}

/*
Return NULL_VALUE if parsedType is not ARRAY. If it is then parse
the receiver and call the values corresponding Slice with the indices
as input arguments.
*/
func (this *parsedValue) Slice(start, end int) (Value, bool) {
	if this.parsedType != ARRAY {
		return NULL_VALUE, false
	}

	return this.parse().Slice(start, end)
}

/*
Return NULL_VALUE if parsedType is not ARRAY. If it is then parse
the receiver and call the values corresponding SliceTail with the
start index as input arguments.
*/
func (this *parsedValue) SliceTail(start int) (Value, bool) {
	if this.parsedType != ARRAY {
		return NULL_VALUE, false
	}

	return this.parse().SliceTail(start)
}

/*
Return the buffer if the parsedType is binary. If not call parse
and then the Descendants method on that value with the input
buffer.
*/
func (this *parsedValue) Descendants(buffer []interface{}) []interface{} {
	if this.parsedType == BINARY {
		return buffer
	}

	return this.parse().Descendants(buffer)
}

/*
Return nil if the parsedType is binary. If not call parse
and then the Fields method on that value.
*/
func (this *parsedValue) Fields() map[string]interface{} {
	if this.parsedType == BINARY {
		return nil
	}

	return this.parse().Fields()
}

/*
It is used to populate the values in the structure. If the parsed value
is nil and the parsedType is binary, panic(error) since an attempt
to parse a non JSON value has been made. If not then create a variable
of type interface, Unmarshal the raw bytes, and add it to it. If there is
an error while unmarshalling, it is an unexpected parse error. If not
populate the value field parsed with the NewVaue of the interface.
The parsed value is finally returned.
*/
func (this *parsedValue) parse() Value {
	if this.parsed == nil {
		if this.parsedType == BINARY {
			panic("Attempt to parse non-JSON value.")
		}

		var p interface{}
		err := json.Unmarshal(this.raw, &p)
		if err != nil {
			panic("Unexpected parse error on valid JSON.")
		}
		this.parsed = NewValue(p)
	}

	return this.parsed
}
