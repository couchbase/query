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
	"encoding/json"
	"strconv"

	jsonpointer "github.com/dustin/go-jsonpointer"
)

/*
A Value with delayed parsing.
*/
type parsedValue struct {
	raw        []byte
	parsedType Type
	parsed     Value
}

func (this *parsedValue) MarshalJSON() ([]byte, error) {
	return this.unwrap().MarshalJSON()
}

func (this *parsedValue) Type() Type {
	return this.parsedType
}

func (this *parsedValue) Actual() interface{} {
	return this.unwrap().Actual()
}

func (this *parsedValue) Equals(other Value) Value {
	return this.unwrap().Equals(other)
}

func (this *parsedValue) Collate(other Value) int {
	return this.unwrap().Collate(other)
}

func (this *parsedValue) Compare(other Value) Value {
	return this.unwrap().Compare(other)
}

func (this *parsedValue) Truth() bool {
	return this.unwrap().Truth()
}

func (this *parsedValue) Copy() Value {
	return this.unwrap().Copy()
}

func (this *parsedValue) CopyForUpdate() Value {
	return this.unwrap().CopyForUpdate()
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

	return this.unwrap().SetField(field, val)
}

/*
Return Unsettable if parsedType is not OBJECT. If it is then parse
the receiver and call the values corresponding UnsetField.
*/
func (this *parsedValue) UnsetField(field string) error {
	if this.parsedType != OBJECT {
		return Unsettable(field)
	}

	return this.unwrap().UnsetField(field)
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

	return this.unwrap().SetIndex(index, val)
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

	return this.unwrap().Slice(start, end)
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

	return this.unwrap().SliceTail(start)
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

	return this.unwrap().Descendants(buffer)
}

func (this *parsedValue) Fields() map[string]interface{} {
	return this.unwrap().Fields()
}

func (this *parsedValue) Successor() Value {
	return this.unwrap().Successor()
}

/*
Delayed parse.
*/
func (this *parsedValue) unwrap() Value {
	if this.parsed == nil {
		if this.parsedType == BINARY {
			this.parsed = binaryValue(this.raw)
		} else {
			var p interface{}
			err := json.Unmarshal(this.raw, &p)
			if err != nil {
				panic("Unexpected parse error on valid JSON.")
			}

			this.parsed = NewValue(p)
		}
	}

	return this.parsed
}
