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
	"math"

	json "github.com/dustin/gojson"
)

type floatValue float64

var ZERO_VALUE = NewValue(0.0)
var ONE_VALUE = NewValue(1.0)

func (this floatValue) Type() int {
	return NUMBER
}

func (this floatValue) Actual() interface{} {
	return float64(this)
}

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

func (this floatValue) Collate(other Value) int {
	switch other := other.(type) {
	case floatValue:
		return int(this - other)
	case *parsedValue:
		return this.Collate(other.parse())
	case *annotatedValue:
		return this.Collate(other.Value)
	default:
		return NUMBER - other.Type()
	}

}

func (this floatValue) Truth() bool {
	return !math.IsNaN(float64(this)) && this != 0
}

func (this floatValue) Copy() Value {
	return this
}

func (this floatValue) CopyForUpdate() Value {
	return this
}

func (this floatValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this floatValue) Field(field string) (Value, bool) {
	return NULL_VALUE, false
}

func (this floatValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this floatValue) UnsetField(field string) error {
	return Unsettable(field)
}

func (this floatValue) Index(index int) (Value, bool) {
	return NULL_VALUE, false
}

func (this floatValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

func (this floatValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}

func (this floatValue) SliceTail(start int) (Value, bool) {
	return NULL_VALUE, false
}

func (this floatValue) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

func (this floatValue) Fields() map[string]interface{} {
	return nil
}
