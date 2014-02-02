//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import ()

type boolValue bool

func (this boolValue) Type() int {
	return BOOLEAN
}

func (this boolValue) Actual() interface{} {
	return bool(this)
}

func (this boolValue) Equals(other Value) bool {
	switch other := other.(type) {
	case boolValue:
		return this == other
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

func (this boolValue) Collate(other Value) int {
	switch other := other.(type) {
	case boolValue:
		if this == other {
			return 0
		} else if !this {
			return -1
		} else {
			return 1
		}
	case *parsedValue:
		return this.Collate(other.parse())
	case *annotatedValue:
		return this.Collate(other.Value)
	default:
		return BOOLEAN - other.Type()
	}

}

func (this boolValue) Copy() Value {
	return this
}

func (this boolValue) CopyForUpdate() Value {
	return this
}

var _FALSE_BYTES = []byte("false")
var _TRUE_BYTES = []byte("true")

func (this boolValue) Bytes() []byte {
	if this {
		return _TRUE_BYTES
	} else {
		return _FALSE_BYTES
	}
}

func (this boolValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this boolValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this boolValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this boolValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}
