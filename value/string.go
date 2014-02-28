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
	json "github.com/dustin/gojson"
)

type stringValue string

func (this stringValue) Type() int {
	return STRING
}

func (this stringValue) Actual() interface{} {
	return string(this)
}

func (this stringValue) Equals(other Value) bool {
	switch other := other.(type) {
	case stringValue:
		return this == other
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

func (this stringValue) Collate(other Value) int {
	switch other := other.(type) {
	case stringValue:
		if this < other {
			return -1
		} else if this > other {
			return 1
		} else {
			return 0
		}
	case *parsedValue:
		return this.Collate(other.parse())
	case *annotatedValue:
		return this.Collate(other.Value)
	default:
		return STRING - other.Type()
	}

}

func (this stringValue) Truth() bool {
	return len(this) > 0
}

func (this stringValue) Copy() Value {
	return this
}

func (this stringValue) CopyForUpdate() Value {
	return this
}

func (this stringValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this stringValue) Field(field string) (Value, bool) {
	return missingField(field), false
}

func (this stringValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this stringValue) Index(index int) (Value, bool) {
	return missingIndex(index), false
}

func (this stringValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

func (this stringValue) Slice(start, end int) (Value, bool) {
	return MISSING_VALUE, false
}
