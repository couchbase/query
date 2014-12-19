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
	"math"
)

/*
Number, represented by floatValue is defined as type float64.
*/
type floatValue float64

/*
The variables ZERO_VALUE and ONE_VALUE are initialized to
0.0 and 1.0 respectively.
*/
var ZERO_VALUE = NewValue(0.0)
var ONE_VALUE = NewValue(1.0)

/*
MarshalJSON casts the method receiver to float64, and uses
the math package functions to check if its NaN, +infinity
or –infinity, in which case it returns a slice of byte
representing that value, else it calls jsons marshal
function on the cast value.
*/
func (this floatValue) MarshalJSON() ([]byte, error) {
	f := float64(this)

	if math.IsNaN(f) {
		return []byte("\"NaN\""), nil
	} else if math.IsInf(f, 1) {
		return []byte("\"+Infinity\""), nil
	} else if math.IsInf(f, -1) {
		return []byte("\"-Infinity\""), nil
	} else {
		if f == -0 {
			f = 0
		}

		return json.Marshal(f)
	}
}

/*
Type Number
*/
func (this floatValue) Type() Type { return NUMBER }

/*
Cast receiver to float64(Go type).
*/
func (this floatValue) Actual() interface{} {
	return float64(this)
}

/*
If other is a floatValue, compare it with the receiver.
If it is a parsedValue or annotated value then call Equals
by parsing other or Values respectively. If it is any other
type we return false.
*/
func (this floatValue) Equals(other Value) bool {
	switch other := other.(type) {
	case floatValue:
		return this == other
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

/*
If other is a floatValue, subtract it from the receiver.
If it is less thatn 0.0 return -1, if greater return 1
and otherwise return 0. For value of type parsedValue and
annotated value call collate again with the value. The
default behavior is to return the position wrt others type.
*/
func (this floatValue) Collate(other Value) int {
	switch other := other.(type) {
	case floatValue:
		result := float64(this - other)
		switch {
		case result < 0.0:
			return -1
		case result > 0.0:
			return 1
		}
		return 0
	case *parsedValue:
		return this.Collate(other.parse())
	case *annotatedValue:
		return this.Collate(other.Value)
	default:
		return int(NUMBER - other.Type())
	}

}

/*
Returns true in the event the receiver is not 0 and it isn’t
a NaN value
*/
func (this floatValue) Truth() bool {
	return !math.IsNaN(float64(this)) && this != 0
}

/*
Return receiver
*/
func (this floatValue) Copy() Value {
	return this
}

/*
Return receiver
*/
func (this floatValue) CopyForUpdate() Value {
	return this
}

/*
Calls missingField.
*/
func (this floatValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

/*
Not valid for NUMBER.
*/
func (this floatValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

/*
Not valid for NUMBER.
*/
func (this floatValue) UnsetField(field string) error {
	return Unsettable(field)
}

/*
Calls missingIndex.
*/
func (this floatValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

/*
Not valid for NUMBER.
*/
func (this floatValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

/*
Returns NULL_VALUE
*/
func (this floatValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns NULL_VALUE
*/
func (this floatValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

/*
Returns the input buffer as is.
*/
func (this floatValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

/*
As number has no fields, return nil.
*/
func (this floatValue) Fields() map[string]interface{} {
	return nil
}
