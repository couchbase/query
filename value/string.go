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
	"fmt"

	"github.com/couchbase/query/util"
)

/*
stringValue is defined as type string.
*/
type stringValue string

/*
Define a value representing an empty string and
assign it to EMPTY_STRING_VALUE.
*/
var EMPTY_STRING_VALUE = NewValue("")

/*
Use built-in JSON string marshalling, which handles special
characters.
*/
func (this stringValue) String() string {
	bytes, err := json.Marshal(string(this))
	if err != nil {
		// We should not get here.
		panic(fmt.Sprintf("Error marshaling Value %v: %v", this, err))
	}
	return string(bytes)
}

/*
Use built-in JSON string marshalling, which handles special
characters.
*/
func (this stringValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(this))
}

/*
Type STRING.
*/
func (this stringValue) Type() Type {
	return STRING
}

/*
Cast receiver to string and return.
*/
func (this stringValue) Actual() interface{} {
	return string(this)
}

/*
If other is type stringValue and is the same as the receiver
return true.
*/
func (this stringValue) Equals(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	case stringValue:
		if this == other {
			return TRUE_VALUE
		}
	}

	return FALSE_VALUE
}

/*
If other is type stringValue, compare with receiver,
if its less than (string comparison) return -1, greater
than return 1, otherwise return 0. For value of type
parsedValue and annotated value call collate again with the
value. The default behavior is to return the position wrt
others type.
*/
func (this stringValue) Collate(other Value) int {
	other = other.unwrap()
	switch other := other.(type) {
	case stringValue:
		if this < other {
			return -1
		} else if this > other {
			return 1
		} else {
			return 0
		}
	default:
		return int(STRING - other.Type())
	}
}

func (this stringValue) Compare(other Value) Value {
	other = other.unwrap()
	switch other := other.(type) {
	case missingValue:
		return other
	case *nullValue:
		return other
	default:
		return NewValue(this.Collate(other))
	}
}

/*
If length of string greater than 0, its a valid string.
Return true.
*/
func (this stringValue) Truth() bool {
	return len(this) > 0
}

/*
Return receiver.
*/
func (this stringValue) Copy() Value {
	return this
}

/*
Return receiver.
*/
func (this stringValue) CopyForUpdate() Value {
	return this
}

/*
Calls missingField.
*/
func (this stringValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Not valid for string.
*/
func (this stringValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Not valid for string.
*/
func (this stringValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
Calls missingIndex.
*/
func (this stringValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Not valid for string.
*/
func (this stringValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

/*
Returns NULL_VALUE
*/
func (this stringValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns NULL_VALUE
*/
func (this stringValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns the input buffer as is.
*/
func (this stringValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

/*
No fields to list. Hence return nil.
*/
func (this stringValue) Fields() map[string]interface{} {
	return nil
}

func (this stringValue) FieldNames(buffer []string) []string {
	return nil
}

/*
Returns the input buffer as is.
*/
func (this stringValue) DescendantPairs(buffer []util.IPair) []util.IPair {
	return buffer
}

/*
Append a low-valued byte to string.
*/
func (this stringValue) Successor() Value {
	return NewValue(string(this) + " ")
}

func (this stringValue) unwrap() Value {
	return this
}
