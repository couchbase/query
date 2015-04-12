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
	"fmt"
)

/*
Missing value is of type string
*/
type missingValue string

/*
Initialized to an empty string cast to missingValue.
*/
var MISSING_VALUE Value = missingValue("")

/*
Returns variable MISING_VALUE.
*/
func NewMissingValue() Value {
	return MISSING_VALUE
}

/*
Ideally, we should never marshal a MISSING value. Hence
_NULL_BYTES is returned.
*/
func (this missingValue) MarshalJSON() ([]byte, error) {
	return _NULL_BYTES, nil
}

/*
Description of which property or index was undefined (if known).
If not known, return a message stating Missing field or index.
*/
func (this missingValue) Error() string {
	if string(this) != "" {
		return fmt.Sprintf("Missing field or index %s.", string(this))
	} else {
		return "Missing field or index."
	}
}

/*
Type MISSING
*/
func (this missingValue) Type() Type {
	return MISSING
}

/*
Returns nil, since this is not a valid Go type.
*/
func (this missingValue) Actual() interface{} {
	return nil
}

/*
Returns false.
*/
func (this missingValue) Equals(other Value) Value {
	return this
}

/*
Returns an integer representing the position of Missing with
respect to the other values type by subtracting them and
casting the result to an integer.
*/
func (this missingValue) Collate(other Value) int {
	return int(MISSING - other.Type())
}

func (this missingValue) Compare(other Value) Value {
	return this
}

/*
As per the N1ql specs the truth-value of a missing evaluates
to a false, and hence the Truth method returns a false.
*/
func (this missingValue) Truth() bool {
	return false
}

/*
Return receiver this.
*/
func (this missingValue) Copy() Value {
	return this
}

/*
Return receiver.
*/
func (this missingValue) CopyForUpdate() Value {
	return this
}

/*
Calls missingField.
*/
func (this missingValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Invalid for missing.
*/
func (this missingValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Invalid for missing.
*/
func (this missingValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
Calls missingIndex.
*/
func (this missingValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Invalid for missing.
*/
func (this missingValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

/*
Returns MISSING_VALUE.
*/
func (this missingValue) Slice(start, end int) (Value, bool) {
	return MISSING_VALUE, false
}

/*
Returns MISSING_VALUE.
*/
func (this missingValue) SliceTail(start int) (Value, bool) {
	return MISSING_VALUE, false
}

/*
Returns the input buffer as is.
*/
func (this missingValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

/*
MISSING has no fields to list. Hence return nil.
*/
func (this missingValue) Fields() map[string]interface{} {
	return nil
}

/*
MISSING is succeeded by NULL.
*/
func (this missingValue) Successor() Value {
	return NULL_VALUE
}

func (this missingValue) unwrap() Value {
	return this
}

/*
Cast input field to missingValue.
*/
func missingField(field string) missingValue {
	return missingValue(field)
}

/*
Cast input index to missingValue after casting to string.
*/
func missingIndex(index int) missingValue {
	return missingValue(string(index))
}
