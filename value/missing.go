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

// Missing value
func NewMissingValue() Value {
	return &_MISSING_VALUE
}

type missingValue struct {
}

var _MISSING_VALUE = missingValue{}

func (this *missingValue) Type() int {
	return MISSING
}

func (this *missingValue) Actual() interface{} {
	return nil
}

func (this *missingValue) Equals(other Value) bool {
	switch other := other.(type) {
	case *missingValue:
		return true
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

func (this *missingValue) Copy() Value {
	return this
}

func (this *missingValue) CopyForUpdate() Value {
	return this
}

var _MISSING_BYTES = []byte("missing")

func (this *missingValue) Bytes() []byte {
	return _MISSING_BYTES
}

func (this *missingValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this *missingValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this *missingValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this *missingValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}
