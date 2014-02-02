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

type sliceValue []interface{}

func (this sliceValue) Type() int {
	return ARRAY
}

func (this sliceValue) Actual() interface{} {
	return ([]interface{})(this)
}

func (this sliceValue) Equals(other Value) bool {
	switch other := other.(type) {
	case sliceValue:
		return arrayEquals(this, other)
	case *listValue:
		return arrayEquals(this, other.actual)
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

func (this sliceValue) Copy() Value {
	return sliceValue(copySlice(this, self))
}

func (this sliceValue) CopyForUpdate() Value {
	return &listValue{copySlice(this, copyForUpdate)}
}

func (this sliceValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this sliceValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this sliceValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this sliceValue) Index(index int) (Value, error) {
	if index >= 0 && index < len(this) {
		return NewValue(this[index]), nil
	}

	// consistent with parsedValue
	return nil, Undefined(index)
}

// NOTE: Slices do NOT extend beyond length.
func (this sliceValue) SetIndex(index int, val interface{}) error {
	if index < 0 || index >= len(this) {
		return Unsettable(index)
	}

	this[index] = val
	return nil
}

type listValue struct {
	actual []interface{}
}

func (this *listValue) Type() int {
	return ARRAY
}

func (this *listValue) Actual() interface{} {
	return this.actual
}

func (this *listValue) Equals(other Value) bool {
	switch other := other.(type) {
	case *listValue:
		return arrayEquals(this.actual, other.actual)
	case sliceValue:
		return arrayEquals(this.actual, other)
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

func (this *listValue) Copy() Value {
	return &listValue{copySlice(this.actual, self)}
}

func (this *listValue) CopyForUpdate() Value {
	return &listValue{copySlice(this.actual, copyForUpdate)}
}

func (this *listValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this *listValue) Field(field string) (Value, error) {
	return nil, Undefined(field)
}

func (this *listValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this *listValue) Index(index int) (Value, error) {
	if index >= 0 && index < len(this.actual) {
		return NewValue(this.actual[index]), nil
	}

	// consistent with parsedValue
	return nil, Undefined(index)
}

func (this *listValue) SetIndex(index int, val interface{}) error {
	if index < 0 {
		return Unsettable(index)
	}

	if index >= len(this.actual) {
		if index < cap(this.actual) {
			this.actual = this.actual[0 : index+1]
		} else {
			act := make([]interface{}, index+1, (index+1)<<1)
			copy(act, this.actual)
			this.actual = act
		}
	}

	this.actual[index] = val
	return nil
}

func arrayEquals(first, second []interface{}) bool {
	if len(first) != len(second) {
		return false
	}

	for i, f := range first {
		if !NewValue(f).Equals(NewValue(second[i])) {
			return false
		}
	}

	return true
}
