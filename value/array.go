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
	case *scopeValue:
		return this.Equals(other.Value)
	case *annotatedValue:
		return this.Equals(other.Value)
	case *parsedValue:
		return this.Equals(other.parse())
	default:
		return false
	}
}

func (this sliceValue) Collate(other Value) int {
	switch other := other.(type) {
	case sliceValue:
		return arrayCollate(this, other)
	case *listValue:
		return arrayCollate(this, other.actual)
	case *scopeValue:
		return this.Collate(other.Value)
	case *annotatedValue:
		return this.Collate(other.Value)
	case *parsedValue:
		return this.Collate(other.parse())
	default:
		return ARRAY - other.Type()
	}
}

func (this sliceValue) Truth() bool {
	return len(this) > 0
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

func (this sliceValue) Field(field string) (Value, bool) {
	return NULL_VALUE, false
}

func (this sliceValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this sliceValue) UnsetField(field string) error {
	return Unsettable(field)
}

func (this sliceValue) Index(index int) (Value, bool) {
	if index < 0 {
		index = len(this) + index
	}

	if index >= 0 && index < len(this) {
		return NewValue(this[index]), true
	}

	return missingIndex(index), false
}

// NOTE: Slices do NOT extend beyond length.
func (this sliceValue) SetIndex(index int, val interface{}) error {
	if index < 0 {
		index = len(this) + index
	}

	if index < 0 || index >= len(this) {
		return Unsettable(index)
	}

	this[index] = val
	return nil
}

func (this sliceValue) Slice(start, end int) (Value, bool) {
	if start < 0 {
		start = len(this) + start
	}

	if end < 0 {
		end = len(this) + end
	}

	if start <= end && start >= 0 && end <= len(this) {
		return NewValue(this[start:end]), true
	}

	return MISSING_VALUE, false
}

func (this sliceValue) SliceTail(start int) (Value, bool) {
	if start < 0 {
		start = len(this) + start
	}

	if start >= 0 {
		return NewValue(this[start:]), true
	}

	return MISSING_VALUE, false
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
	case *scopeValue:
		return this.Equals(other.Value)
	case *annotatedValue:
		return this.Equals(other.Value)
	case *parsedValue:
		return this.Equals(other.parse())
	default:
		return false
	}
}

func (this *listValue) Collate(other Value) int {
	switch other := other.(type) {
	case *listValue:
		return arrayCollate(this.actual, other.actual)
	case sliceValue:
		return arrayCollate(this.actual, other)
	case *scopeValue:
		return this.Collate(other.Value)
	case *annotatedValue:
		return this.Collate(other.Value)
	case *parsedValue:
		return this.Collate(other.parse())
	default:
		return ARRAY - other.Type()
	}
}

func (this *listValue) Truth() bool {
	return len(this.actual) > 0
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

func (this *listValue) Field(field string) (Value, bool) {
	return NULL_VALUE, false
}

func (this *listValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this *listValue) UnsetField(field string) error {
	return Unsettable(field)
}

func (this *listValue) Index(index int) (Value, bool) {
	if index < 0 {
		index = len(this.actual) + index
	}

	if index >= 0 && index < len(this.actual) {
		return NewValue(this.actual[index]), true
	}

	return missingIndex(index), false
}

func (this *listValue) SetIndex(index int, val interface{}) error {
	if index < 0 {
		index = len(this.actual) + index
	}

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

func (this *listValue) Slice(start, end int) (Value, bool) {
	if start < 0 {
		start = len(this.actual) + start
	}

	if end < 0 {
		end = len(this.actual) + end
	}

	if start <= end && start >= 0 && end <= len(this.actual) {
		return NewValue(this.actual[start:end]), true
	}

	return MISSING_VALUE, false
}

func (this *listValue) SliceTail(start int) (Value, bool) {
	if start < 0 {
		start = len(this.actual) + start
	}

	if start >= 0 {
		return NewValue(this.actual[start:]), true
	}

	return MISSING_VALUE, false
}

func arrayEquals(array1, array2 []interface{}) bool {
	if len(array1) != len(array2) {
		return false
	}

	for i, item1 := range array1 {
		if !NewValue(item1).Equals(NewValue(array2[i])) {
			return false
		}
	}

	return true
}

// this code originally taken from walrus
// https://github.com/couchbaselabs/walrus
func arrayCollate(array1, array2 []interface{}) int {
	for i, item1 := range array1 {
		if i >= len(array2) {
			return 1
		}

		if cmp := NewValue(item1).Collate(NewValue(array2[i])); cmp != 0 {
			return cmp
		}
	}

	return len(array1) - len(array2)
}

func copySlice(source []interface{}, copier copyFunc) []interface{} {
	if source == nil {
		return nil
	}

	result := make([]interface{}, len(source))
	for i, v := range source {
		result[i] = copier(v)
	}

	return result
}
